package utils

import (
	"fmt"
	"image"
	"image/color"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"gocv.io/x/gocv"
)

// DecodeQRCodeZXing converts a gocv.Mat to an image.Image,
// then uses gozxing to detect and decode a QR code.
// It returns the decoded text or an error.
func DecodeQRCodeZXing(mat gocv.Mat) (string, error) {
	// Convert gocv.Mat to image.Image.
	img, err := mat.ToImage()
	if err != nil {
		return "", fmt.Errorf("failed to convert Mat to image: %v", err)
	}

	// Create a LuminanceSource from the image.
	source := gozxing.NewLuminanceSourceFromImage(img)
	// Create a BinaryBitmap using HybridBinarizer.
	bitmap, err := gozxing.NewBinaryBitmap(gozxing.NewHybridBinarizer(source))
	if err != nil {
		return "", fmt.Errorf("failed to create binary bitmap: %v", err)
	}

	// Create a QR code reader.
	reader := qrcode.NewQRCodeReader()
	result, err := reader.Decode(bitmap, nil)
	if err != nil {
		return "", err
	}
	return result.GetText(), nil
}

// ProcessQRRegion extracts a subregion defined by rect from the given image,
// converts it to grayscale, decodes the QR code in that region, and if successful,
// draws the rectangle and decoded text on the original image.
// It returns the decoded QR text or an error.
func ProcessQRRegion(img *gocv.Mat, rect image.Rectangle) (string, error) {
	// Extract the sub-mat from the original image.
	subMat := img.Region(rect)
	defer subMat.Close()

	// Convert the sub-mat to grayscale.
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(subMat, &gray, gocv.ColorBGRToGray)

	// Decode the QR code using the utility function.
	qrText, _ := DecodeQRCodeZXing(gray)

	// Draw the rectangle on the original image.
	gocv.Rectangle(img, rect, color.RGBA{0, 255, 0, 0}, 2)
	// Put the decoded QR text above the rectangle.
	ptText := image.Pt(rect.Min.X, rect.Max.Y+10)
	gocv.PutText(img, qrText, ptText, gocv.FontHersheyPlain, 1.2, color.RGBA{0, 0, 255, 0}, 2)

	return qrText, nil
}
