//go:build linux

package tray

import (
	"image"
	"image/color"
)

// pixmap is the (iiay) icon structure of the StatusNotifierItem spec: ARGB32
// pixels in network byte order.
type pixmap struct {
	Width  int32
	Height int32
	Data   []byte
}

// Thumbnail scales img to a size×size square after cropping its transparent
// borders, so the artwork fills small icon slots.
func Thumbnail(img image.Image, size int) *image.NRGBA {
	return scaleSquare(img, opaqueSquare(img), size)
}

func iconPixmaps(img image.Image, sizes ...int) []pixmap {
	src := opaqueSquare(img)
	out := make([]pixmap, 0, len(sizes))
	for _, size := range sizes {
		out = append(out, toPixmap(scaleSquare(img, src, size)))
	}
	return out
}

func toPixmap(img *image.NRGBA) pixmap {
	b := img.Bounds()
	data := make([]byte, 0, b.Dx()*b.Dy()*4)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.NRGBAAt(x, y)
			data = append(data, c.A, c.R, c.G, c.B)
		}
	}
	return pixmap{Width: int32(b.Dx()), Height: int32(b.Dy()), Data: data}
}

// opaqueSquare returns the smallest square centred on the non-transparent
// content of img. It may extend past img's bounds.
func opaqueSquare(img image.Image) image.Rectangle {
	b := img.Bounds()
	var content image.Rectangle
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a != 0 {
				content = content.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	if content.Empty() {
		content = b
	}
	side := max(content.Dx(), content.Dy())
	origin := image.Pt(content.Min.X-(side-content.Dx())/2, content.Min.Y-(side-content.Dy())/2)
	return image.Rectangle{Min: origin, Max: origin.Add(image.Pt(side, side))}
}

// scaleSquare box-filters the square src region of img down to size×size,
// averaging in premultiplied space so transparent pixels do not darken edges.
func scaleSquare(img image.Image, src image.Rectangle, size int) *image.NRGBA {
	bounds := img.Bounds()
	side := src.Dx()
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	for dy := 0; dy < size; dy++ {
		y0 := src.Min.Y + dy*side/size
		y1 := max(src.Min.Y+(dy+1)*side/size, y0+1)
		for dx := 0; dx < size; dx++ {
			x0 := src.Min.X + dx*side/size
			x1 := max(src.Min.X+(dx+1)*side/size, x0+1)

			var r, g, b, a, n uint64
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					n++
					if !image.Pt(x, y).In(bounds) {
						continue
					}
					pr, pg, pb, pa := img.At(x, y).RGBA()
					r += uint64(pr)
					g += uint64(pg)
					b += uint64(pb)
					a += uint64(pa)
				}
			}
			if a == 0 {
				continue
			}
			dst.SetNRGBA(dx, dy, color.NRGBA{
				R: uint8(r * 0xffff / a >> 8),
				G: uint8(g * 0xffff / a >> 8),
				B: uint8(b * 0xffff / a >> 8),
				A: uint8(a / n >> 8),
			})
		}
	}
	return dst
}
