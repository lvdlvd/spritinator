/*
The Spritinator reads a set of images (jpeg, png, ) listed on the commandline
composes them into one big .png file, and produces a json object mapping
the original names to x,y offsets and widht and height in the sprite.

The placement algorithm is rather simplistic: a ±sqrt(n) x sqrt(n) grid
of maxwidth x maxheight rectangles.
*/
package main

import (
	"encoding/json"
	"flag"
	"image"
	"log"
	"math"
	"os"
	"sort"
	"strings"

	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
)

var (
	out = flag.String("o", "sprite", "Base name of the .png and .json files to write.")
	pfx = flag.String("pfx", "", "Prefix toc entry names with this.")
	ps  = flag.Int("s", 0, "number of directory components to skip in the toc entries.")
)

const padding = 2 // 2 pixels between images

func tocPath(s string) string {
	if *ps == 0 {
		return *pfx + s
	}
	parts := strings.Split(s, string(os.PathSeparator))
	if len(parts) <= *ps {
		return *pfx + s
	}
	return *pfx + strings.Join(parts[*ps:], string(os.PathSeparator))
}

type item struct {
	Name       string `json:"-"`
	X, Y, W, H int
	img        image.Image
}

type byHW []*item

func (b byHW) Len() int      { return len(b) }
func (b byHW) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byHW) Less(i, j int) bool {
	if b[i].H != b[j].H {
		return b[i].H < b[j].H
	}
	if b[i].W != b[j].W {
		return b[i].W < b[j].W
	}
	return b[i].Name < b[j].Name
}

func main() {
	flag.Parse()

	var images []*item
	var maxw, maxh int

	for _, v := range flag.Args() {
		f, err := os.Open(v)
		if err != nil {
			log.Print(err)
			continue
		}
		src, format, err := image.Decode(f)
		f.Close()
		if err != nil {
			log.Printf("%s: %s", v, err)
			continue
		}

		b := src.Bounds()
		p := tocPath(v)
		log.Printf("%s: %s %dx%d", p, format, b.Dx(), b.Dy())

		images = append(images, &item{p, 0, 0, b.Dx(), b.Dy(), src})

		if maxw < b.Dx() {
			maxw = b.Dx()
		}
		if maxh < b.Dy() {
			maxh = b.Dy()
		}
	}

	log.Printf("Read %d of %d images.", len(images), len(flag.Args()))
	n := int(math.Sqrt(float64(len(images))) + 1)
	m := (len(images) + n - 1) / n
	if n*m < len(images) {
		panic("lvd is an idiot")
	}
	ow, oh := n*maxw+(n-1)*padding, m*maxh+(m-1)*padding

	log.Printf("Generating %dx%d grid of %dx%d with %d padding = %dx%d", n, m, maxw, maxh, padding, ow, oh)

	sort.Sort(byHW(images))
	img := image.NewRGBA(image.Rect(0, 0, ow, oh))

	byname := make(map[string]*item)
	for i, v := range images {
		k, l := i/n, i%n
		images[i].X, images[i].Y = k*(maxw+padding), l*(maxh+padding)
		dp := image.Point{images[i].X, images[i].Y}
		r := image.Rectangle{dp, dp.Add(v.img.Bounds().Size())}
		draw.Draw(img, r, v.img, v.img.Bounds().Min, draw.Src)
		byname[v.Name] = v
	}

	f, err := os.Create(*out + ".png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}

	if err := json.NewEncoder(os.Stdout).Encode(byname); err != nil {
		log.Fatal(err)
	}
}
