package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, sourceFile); err != nil {
		return err
	}

	return dstFile.Sync()
}

func getIPFromRequest(r *http.Request) string {
	// Attempt to retrieve the IP from the X-Forwarded-For header
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		// Fallback to RemoteAddr if no proxy is involved
		var err error
		ip, _, err = net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return ""
		}
	} else {
		// X-Forwarded-For can contain multiple IPs. Take the first one.
		splitIPs := strings.Split(ip, ",")
		ip = strings.TrimSpace(splitIPs[0])
	}
	return ip
}

func writeOnJPG(imagePath string, text string) {
	// Open the input file
	inFile, err := os.Open(imagePath)
	if err != nil {
		panic(err)
	}
	defer inFile.Close()

	// Decode the image
	img, err := jpeg.Decode(inFile)
	if err != nil {
		panic(err)
	}

	// Add label to the image
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	imgSet := image.NewRGBA(bounds)
	draw.Draw(imgSet, bounds, img, bounds.Min, draw.Src)

	// Create a point to start drawing the text
	face := basicfont.Face7x13
	textWidth := len(text) * face.Advance // Assuming monospaced font for simple calculation
	x := (width - textWidth) / 2          // Center the text horizontally
	y := height/2 + face.Height/2

	// Set the text color and position
	col := color.RGBA{0, 0, 0, 255}
	point := fixed.Point26_6{
		X: fixed.Int26_6(x * 64),
		Y: fixed.Int26_6(y * 64),
	}

	// Draw the label on the image
	d := &font.Drawer{
		Dst:  imgSet,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(text)

	// Create the output file
	outFile, err := os.Create(imagePath)
	if err != nil {
		panic(err)
	}
	defer outFile.Close()

	// Encode the image to the file
	if err := jpeg.Encode(outFile, imgSet, nil); err != nil {
		panic(err)
	}
}

func requestImageHandler(w http.ResponseWriter, r *http.Request) {
	// Get the image name from the URL
	imageName := r.URL.Query().Get("imageName")

	// Get the IP
	ip := getIPFromRequest(r)

	// Copy File
	srcPath := filepath.Join("images", imageName+".jpg")
	dstPath := filepath.Join("images", "generated", fmt.Sprintf("%s/%s.jpg", imageName, ip))

	copyFile(srcPath, dstPath)

	writeOnJPG(dstPath, ip)

	// Serve the generated image
	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeFile(w, r, dstPath)
}

func main() {
	fmt.Println("Server is starting")

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	port := os.Getenv("PORT")

	fmt.Println("Server is starting on port " + port)

	//HTTP Router
	http.HandleFunc("GET /image/{imageName}", requestImageHandler)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}

	// Print to terminal that the server is running
	fmt.Println("Server is running on port " + port)
}
