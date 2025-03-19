package utils

import (
	"fmt"
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

// ProcessHorizontalSections takes an image pointer, a rectangular region (assumed to be horizontal),
// and a number of sections to divide that region into.
// It counts the dark pixels (intensity < darkThreshold) in each section and, if one section has significantly more dark pixels
// than the others, returns its 1-based index; otherwise, it returns 0.
// It also draws the rectangle and vertical dividing lines on the original image and writes the standout section index.
func ProcessHorizontalSections(img *gocv.Mat, rect image.Rectangle, numSections int) (int, error) {
	// Extract the sub-mat from the given rectangle.
	subMat := img.Region(rect)
	defer subMat.Close()

	// Convert the sub-mat to grayscale.
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(subMat, &gray, gocv.ColorBGRToGray)

	// Define parameters.
	const darkThreshold = 100.0 // pixel intensities below this are considered "dark"
	const thresholdFactor = 0.5 // 50% higher than the average dark pixel count is considered significant

	width := gray.Cols()
	height := gray.Rows()
	if numSections <= 0 || width == 0 || height == 0 {
		return 0, fmt.Errorf("invalid input dimensions or numSections")
	}

	// Determine the width of each section.
	sectionWidth := float32(width) / float32(numSections)

	// Count dark pixels for each section.
	darkCounts := make([]int, numSections)
	totalCount := 0

	for i := 0; i < numSections; i++ {
		// Calculate ROI for this section.
		xStart := int(float32(i) * sectionWidth)
		xEnd := xStart + int(sectionWidth)
		// Ensure the last section takes any remainder.
		if i == numSections-1 {
			xEnd = width
		}
		roi := image.Rect(xStart, 0, xEnd, height)
		sectionMat := gray.Region(roi)

		// Apply a threshold: use THRESH_BINARY_INV so that dark pixels become white.
		threshMat := gocv.NewMat()
		gocv.Threshold(sectionMat, &threshMat, darkThreshold, 255, gocv.ThresholdBinaryInv)
		count := gocv.CountNonZero(threshMat)
		darkCounts[i] = count
		totalCount += count

		sectionMat.Close()
		threshMat.Close()
	}

	avg := float64(totalCount) / float64(numSections)
	maxCount := 0
	maxIndex := -1
	for i, count := range darkCounts {
		if count > maxCount {
			maxCount = count
			maxIndex = i
		}
	}

	// Decide if a section stands out.
	// (If the maximum dark count is more than (1+thresholdFactor) times the average, we consider it significant.)
	standout := 0 // 0 means no standout
	if avg == 0 {
		if maxCount > 0 {
			standout = maxIndex // use 1-based indexing
		}
	} else if float64(maxCount) > (1.0+thresholdFactor)*avg {
		standout = maxIndex
	}

	// Draw the original rectangle on the image.
	gocv.Rectangle(img, rect, color.RGBA{0, 255, 0, 0}, 2)

	// Draw vertical lines to mark the section boundaries.
	for i := 1; i < numSections; i++ {
		x := rect.Min.X + int(float32(i)*sectionWidth)
		pt1 := image.Pt(x, rect.Min.Y)
		pt2 := image.Pt(x, rect.Max.Y)
		gocv.Line(img, pt1, pt2, color.RGBA{255, 0, 0, 0}, 1)
	}

	// Draw the standout section index as text above the rectangle.
	text := fmt.Sprintf("Standout: %d", standout)
	ptText := image.Pt(rect.Min.X+200, rect.Min.Y-10)
	gocv.PutText(img, text, ptText, gocv.FontHersheyPlain, 1.2, color.RGBA{0, 0, 255, 0}, 2)

	return standout, nil
}
