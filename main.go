package main

import (
	"fmt"
	"html/template"
	"image"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"main/utils"

	"gocv.io/x/gocv"
)

// Product holds the product name and its count.
type Product struct {
	Name  string
	Value int
}

// DB_Type holds the inventory of products.
type DB_Type struct {
	mu    sync.Mutex
	items map[string]Product
}

// inc increments the product's value by a given amount.
// If the product does not exist, it is created with a default name equal to its key.
func (db *DB_Type) inc(key string, amount int) {
	db.mu.Lock()
	defer db.mu.Unlock()
	if prod, exists := db.items[key]; exists {
		prod.Value += amount
		db.items[key] = prod
	} else {
		db.items[key] = Product{Name: key, Value: amount}
	}
}

// updateName updates the product's name.
func (db *DB_Type) updateName(key, newName string) {
	db.mu.Lock()
	defer db.mu.Unlock()
	if prod, exists := db.items[key]; exists {
		prod.Name = newName
		db.items[key] = prod
	}
}

var db = DB_Type{items: map[string]Product{}}

// Parse HTML templates.
var (
	uploadTemplate    = template.Must(template.ParseFiles("templates/upload.html"))
	dashboardTemplate = template.Must(template.ParseFiles("templates/dashboard.html"))
)

func main() {
	// Frontend routes.
	http.HandleFunc("/upload", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			HandleUploadPage(w, req)
		case http.MethodPost:
			HandleUpload(w, req)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/dashboard", HandleDashboard)
	http.HandleFunc("/update", HandleUpdateInventory)
	http.HandleFunc("/updateName", HandleUpdateName)
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/dashboard", http.StatusSeeOther)
	})

	fmt.Println("Server started on port 3000...")
	if err := http.ListenAndServe(":3000", nil); err != nil {
		log.Fatal("Server error: ", err)
	}
}

// HandleUploadPage renders the file upload page.
func HandleUploadPage(w http.ResponseWriter, req *http.Request) {
	if err := uploadTemplate.Execute(w, nil); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}
}

// HandleUpload handles the file upload and calls DecodeDocument.
func HandleUpload(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	err := req.ParseMultipartForm(10 << 20) // up to 10 MB
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	file, _, err := req.FormFile("uploadFile")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Save the uploaded file to a temporary file.
	tempFile, err := os.CreateTemp("", "upload-*.png")
	if err != nil {
		http.Error(w, "Cannot create temporary file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name())

	_, err = io.Copy(tempFile, file)
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}
	tempFile.Close()

	// Process the image to update the inventory.
	DecodeDocument(tempFile.Name())

	// Redirect to the dashboard.
	http.Redirect(w, req, "/dashboard", http.StatusSeeOther)
}

// HandleDashboard renders the dashboard with current inventory.
func HandleDashboard(w http.ResponseWriter, req *http.Request) {
	data := struct {
		Inventory map[string]Product
	}{
		Inventory: db.items,
	}
	if err := dashboardTemplate.Execute(w, data); err != nil {
		http.Error(w, "Error rendering dashboard", http.StatusInternalServerError)
	}
}

// HandleUpdateInventory handles incrementing or decrementing product value.
func HandleUpdateInventory(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	key := req.FormValue("key")
	action := req.FormValue("action")

	// Use delta of 1 for increment, -1 for decrement.
	delta := 1
	if action == "dec" {
		delta = -1
	}
	db.inc(key, delta)
	http.Redirect(w, req, "/dashboard", http.StatusSeeOther)
}

// HandleUpdateName updates the product name based on the form submission.
func HandleUpdateName(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	key := req.FormValue("key")
	newName := req.FormValue("name")
	db.updateName(key, newName)
	http.Redirect(w, req, "/dashboard", http.StatusSeeOther)
}

// DecodeDocument processes the image file, decodes the QR code and bubble regions,
// and updates the inventory. In the loop, the QR code denotes the product key,
// the first bubble section gives the tens digit and the second bubble section gives the ones digit.
func DecodeDocument(inputImage string) {
	// Read the original image in color.
	img := gocv.IMRead(inputImage, gocv.IMReadColor)
	if img.Empty() {
		fmt.Printf("Error reading image: %s\n", inputImage)
		return
	}
	defer img.Close()

	// Loop to process multiple products in the image.
	for i := range 21 {
		offset := int(float32(i) * 83.47)

		// Process product key QR region.
		keyRect := image.Rect(450, 540+offset, 515, 605+offset)
		key, err := utils.ProcessQRRegion(&img, keyRect)
		if err != nil {
			fmt.Printf("QR code not detected for key at offset %d: %v\n", offset, err)
			continue
		}

		if key == "" {
			continue
		}

		// Process tens bubble region.
		tensRect := image.Rect(534, 541+offset, 951, 576+offset)
		tens, err := utils.ProcessHorizontalSections(&img, tensRect, 10)
		if err != nil {
			fmt.Printf("Error processing horizontal sections (tens) at offset %d: %v\n", offset, err)
			continue
		}

		// Process ones bubble region.
		onesRect := image.Rect(980, 541+offset, 1395, 576+offset)
		ones, err := utils.ProcessHorizontalSections(&img, onesRect, 10)
		if err != nil {
			fmt.Printf("Error processing horizontal sections (ones) at offset %d: %v\n", offset, err)
			continue
		}

		// Calculate the decoded count.
		count := tens*10 + ones

		// Update inventory.
		if count != 0 {
			db.inc(key, count)
			fmt.Printf("Updated inventory: key: %s, name: %s, new count: %d (added %d)\n", key, db.items[key].Name, db.items[key].Value, count)
		}

	}
	// Optionally write out the image for debugging; not served to the client.
	gocv.IMWrite("example.png", img)
}
