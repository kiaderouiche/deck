// vgdeck: slide decks for OpenVG
package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/ajstarks/deck"
	"github.com/ajstarks/openvg"
	"os"
	"strings"
	"time"
)

// dodeck sets up the graphics environment and kicks off the interaction
func dodeck(filename string, pausetime time.Duration, slidenum int, gp float64) {
	w, h := openvg.Init()
	openvg.Background(0, 0, 0)
	if pausetime == 0 {
		interact(filename, w, h, slidenum, gp)
	} else {
		loop(filename, w, h, pausetime)
	}
	openvg.Finish()
}

// interact controls the display of the deck
func interact(filename string, w, h, slidenum int, gp float64) {
	openvg.SaveTerm()
	defer openvg.RestoreTerm()
	var d deck.Deck
	var err error
	d, err = deck.Read(filename, w, h)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	openvg.RawTerm()
	r := bufio.NewReader(os.Stdin)
	lastslide := len(d.Slide) - 1
	if slidenum > lastslide {
		slidenum = lastslide
	}
	if slidenum < 0 {
		slidenum = 0
	}
	n := slidenum
	xray := 1
	initial := 0
	// respond to keyboard commands, 'q' to exit
	for cmd := byte('0'); cmd != 'q'; cmd = readcmd(r) {
		switch cmd {
		// read/reload
		case 'r', 18: // r, Ctrl-R
			d, err = deck.Read(filename, w, h)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				return
			}
			openvg.Background(0, 0, 0)
			xray = 1
			showslide(d, n)

		// save slide
		case 's', 19: // s, Ctrl-S
			openvg.SaveEnd(fmt.Sprintf("%s-slide-%04d", filename, n))

		// first slide
		case '0', '1', 1, '^': // 0,1,Ctrl-A,^
			initial++
			if initial == 1 {
				n = slidenum
			} else {
				n = 0
			}
			showslide(d, n)

		// last slide
		case '*', 5, '$': // *, Crtl-E, $
			n = lastslide
			showslide(d, n)

		// next slide
		case '+', 'n', '\n', ' ', '\t', 14, 27: // +,n,newline,space,tab,Crtl-N
			n++
			if n > lastslide {
				n = 0
			}
			showslide(d, n)

		// previous slide
		case '-', 'p', 8, 16, 127: // -,p,Backspace,Ctrl-P,Del
			n--
			if n < 0 {
				n = lastslide
			}
			showslide(d, n)

		// x-ray
		case 'x', 24: // x, Ctrl-X
			xray++
			showslide(d, n)
			if xray%2 == 0 {
				showgrid(d, n, gp)
			}

		// search
		case '/', 6: // slash, Ctrl-F
			openvg.RestoreTerm()
			searchterm, serr := r.ReadString('\n')
			openvg.RawTerm()
			if serr != nil {
				continue
			}
			if len(searchterm) > 2 {
				ns := deck.Search(d, searchterm[0:len(searchterm)-1])
				if ns >= 0 {
					showslide(d, ns)
					n = ns
				}
			}
		}
	}
}

// loop through slides with a pause
func loop(filename string, w, h int, n time.Duration) {
	openvg.SaveTerm()
	defer openvg.RestoreTerm()
	var d deck.Deck
	var err error
	d, err = deck.Read(filename, w, h)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	openvg.RawTerm()
	r := bufio.NewReader(os.Stdin)
	// respond to keyboard commands, 'q' to exit
	for {
		for i := 0; i < len(d.Slide); i++ {
			cmd := readcmd(r)
			if cmd == 'q' {
				return
			}
			showslide(d, i)
			time.Sleep(n)
		}
	}
}
// pct computes percentages
func pct(p float64, m float64) float64 {
	return (p / 100.0) * m
}

// showgrid xrays a slide
func showgrid(d deck.Deck, n int, pct float64) {
	w := float64(d.Canvas.Width)
	h := float64(d.Canvas.Height)
	fs := (w / 100) // labels are 1% of the width
	xpct := (pct / 100.0) * w
	ypct := (pct / 100.0) * h

	openvg.StrokeColor("lightgray", 0.5)
	openvg.StrokeWidth(3)

	// horizontal gridlines
	xl := pct
	for x := xpct; x <= w; x += xpct {
		openvg.Line(x, 0, x, h)
		openvg.Text(x, pct, fmt.Sprintf("%.0f%%", xl), "sans", int(fs))
		xl += pct
	}

	// vertical gridlines
	yl := pct
	for y := ypct; y <= h; y += ypct {
		openvg.Line(0, y, w, y)
		openvg.Text(pct, y, fmt.Sprintf("%.0f%%", yl), "sans", int(fs))
		yl += pct
	}

	// show boundary and location of images
	if n < 0 || n > len(d.Slide) {
		return
	}
	for _, im := range d.Slide[n].Image {
		x := (im.Xp / 100) * w
		y := (im.Yp / 100) * h
		iw := float64(im.Width)
		ih := float64(im.Height)
		openvg.FillRGB(127, 0, 0, 0.3)
		openvg.Circle(x, y, fs)
		openvg.FillRGB(255, 0, 0, 0.1)
		openvg.Rect(x-iw/2, y-ih/2, iw, ih)
	}
	openvg.End()
}

//showtext displays text
func showtext(x, y float64, s, align, font string, fs float64) {
	fontsize := int(fs)
	switch align {
	case "center", "middle", "mid":
		openvg.TextMid(x, y, s, font, fontsize)
	case "right", "end":
		openvg.TextEnd(x, y, s, font, fontsize)
	default:
		openvg.Text(x, y, s, font, fontsize)
	}
}

// showlide displays slides
func showslide(d deck.Deck, n int) {
	if n < 0 || n > len(d.Slide)-1 {
		return
	}
	slide := d.Slide[n]
	if slide.Bg == "" {
		slide.Bg = "white"
	}
	if slide.Fg == "" {
		slide.Fg = "black"
	}
	openvg.Start(d.Canvas.Width, d.Canvas.Height)
	cw := float64(d.Canvas.Width)
	ch := float64(d.Canvas.Height)
	openvg.FillColor(slide.Bg)
	openvg.Rect(0, 0, cw, ch)

	var x, y, fs float64

	// every image in the slide
	for _, im := range slide.Image {
		x = (im.Xp / 100) * cw
		y = (im.Yp / 100) * ch
		openvg.Image(x-float64(im.Width/2), y-float64(im.Height/2), im.Width, im.Height, im.Name)
		if len(im.Caption) > 0 {
			capfs := deck.Pwidth(im.Sp, cw, cw/100)
			if im.Font == "" {
				im.Font = "sans"
			}
			if im.Color == "" {
				openvg.FillColor(slide.Fg)
			} else {
				openvg.FillColor(im.Color)
			}
			if im.Align == "" {
				im.Align = "center"
			}
			switch im.Align {
			case "left", "start":
				x -= float64(im.Width / 2)
			case "right", "end":
				x += float64(im.Width / 2)
			}
			showtext(x, y-((float64(im.Height)/2)+(capfs*2.0)), im.Caption, im.Align, im.Font, capfs)
		}
	}

	// every graphic on the slide

	const defaultColor = "rgb(127,127,127)"
	// line
	for _, line := range slide.Line {
		if line.Color == "" {
			line.Color = defaultColor
		}
		x1, y1, sw := deck.Dimen(d.Canvas, line.Xp1, line.Yp1, line.Sp)
		x2, y2, _ := deck.Dimen(d.Canvas, line.Xp2, line.Yp2, 0)
		openvg.StrokeColor(line.Color)
		if sw == 0 {
			sw = 2.0
		}
		openvg.StrokeWidth(sw)
		openvg.StrokeColor(line.Color)
		fmt.Fprintf(os.Stderr, "x1=%.2f y1=%.2f x2=%.2f y2=%.2f\n", x1, y1, x2, y2)
		openvg.Line(x1, y1, x2, y2)
		openvg.StrokeWidth(0)
	}
	// ellipse
	for _, ellipse := range slide.Ellipse {
		x, y, _ = deck.Dimen(d.Canvas, ellipse.Xp, ellipse.Yp, 0)
		w := pct(ellipse.Wp, cw)
		h := pct(ellipse.Hp, cw)
		if ellipse.Color == "" {
			ellipse.Color = defaultColor
		}
		openvg.FillColor(ellipse.Color)
		openvg.Ellipse(x, y, w, h)
	}
	// rect
	for _, rect := range slide.Rect {
		x, y, _ = deck.Dimen(d.Canvas, rect.Xp, rect.Yp, 0)
		w := pct(rect.Wp, cw)
		h := pct(rect.Hp, cw)
		if rect.Color == "" {
			rect.Color = defaultColor
		}
		openvg.FillColor(rect.Color)
		openvg.Rect(x, y, w, h)
	}
	// curve
	for _, curve := range slide.Curve {
		if curve.Color == "" {
			curve.Color = defaultColor
		}
		x1, y1, sw := deck.Dimen(d.Canvas, curve.Xp1, curve.Yp1, curve.Sp)
		x2, y2, _ := deck.Dimen(d.Canvas, curve.Xp2, curve.Yp2, 0)
		x3, y3, _ := deck.Dimen(d.Canvas, curve.Xp3, curve.Yp3, 0)
		openvg.StrokeColor(curve.Color)
		openvg.FillColor(slide.Bg)
		if sw == 0 {
			sw = 1.0
		}
		openvg.StrokeWidth(sw)
		openvg.Qbezier(x1, y1, x2, y2, x3, y3)
		openvg.StrokeWidth(0)
	}

	// arc 
	for _, arc := range slide.Arc {
		if arc.Color == "" {
			arc.Color = defaultColor
		}
		ax, ay, sw := deck.Dimen(d.Canvas, arc.Xp, arc.Yp, arc.Sp)
		w := pct(arc.Wp, cw)
		h := pct(arc.Hp, cw)
		openvg.StrokeColor(arc.Color)
		openvg.FillColor(slide.Bg)
		if sw == 0 {
			sw = 2.0
		}
		openvg.StrokeWidth(sw)
		openvg.Arc(ax, ay, w, h, arc.A1, arc.A2)
		openvg.StrokeWidth(0)
	}
	openvg.FillColor(slide.Fg)
	// every list in the slide
	var offset float64
	const blinespacing = 2.0
	for _, l := range slide.List {
		if l.Font == "" {
			l.Font = "sans"
		}
		x, y, fs = deck.Dimen(d.Canvas, l.Xp, l.Yp, l.Sp)
		if l.Type == "bullet" {
			offset = 1.2 * fs
		} else {
			offset = 0
		}
		if l.Color == "" {
			openvg.FillColor(slide.Fg)
		} else {
			openvg.FillColor(l.Color)
		}
		// every list item
		for ln, li := range l.Li {
			if l.Type == "bullet" {
				boffset := fs / 2
				openvg.Rect(x, y+boffset/2, boffset, boffset)
				//openvg.Circle(x, y+boffset, boffset)
			}
			if l.Type == "number" {
				li = fmt.Sprintf("%d. ", ln+1) + li
			}
			showtext(x+offset, y, li, l.Align, l.Font, fs)
			y -= fs * blinespacing
		}
	}
	openvg.FillColor(slide.Fg)

	// every text in the slide
	const linespacing = 1.8
	for _, t := range slide.Text {
		if t.Font == "" {
			t.Font = "sans"
		}
		x, y, fs = deck.Dimen(d.Canvas, t.Xp, t.Yp, t.Sp)
		td := strings.Split(t.Tdata, "\n")
		if t.Type == "code" {
			t.Font = "mono"
			tdepth := ((fs * linespacing) * float64(len(td))) + fs
			openvg.FillColor("rgb(240,240,240)")
			openvg.Rect(x-20, y-tdepth+(fs*linespacing), deck.Pwidth(t.Wp, cw, cw-x-20), tdepth)
		}
		if t.Color == "" {
			openvg.FillColor(slide.Fg)
		} else {
			openvg.FillColor(t.Color)
		}
		if t.Type == "block" {
			textwrap(x, y, deck.Pwidth(t.Wp, cw, cw/2), t.Tdata, t.Font, fs, fs*linespacing, 0.3)
		} else {
			// every text line
			for _, txt := range td {
				showtext(x, y, txt, t.Align, t.Font, fs)
				y -= (fs * linespacing)
			}
		}
	}
	openvg.FillColor(slide.Fg)
	openvg.End()
}

// whitespace determines if a rune is whitespace
func whitespace(r rune) bool {
	return r == ' ' || r == '\n' || r == '\t' || r == '-'
}

// textwrap draws text at location, wrapping at the specified width
func textwrap(x, y, w float64, s string, font string, fs, leading, factor float64) {
	size := int(fs)
	wordspacing := openvg.TextWidth("m", font, size)
	words := strings.FieldsFunc(s, whitespace)
	xp := x
	yp := y
	edge := x + w
	for _, s := range words {
		tw := openvg.TextWidth(s, font, size)
		openvg.Text(xp, yp, s, font, size)
		xp += tw + (wordspacing * factor)
		if xp > edge {
			xp = x
			yp -= leading
		}
	}
}

// readcmd reads interaction commands
func readcmd(r *bufio.Reader) byte {
	s, err := r.ReadByte()
	if err != nil {
		return 'e'
	}
	return s
}

// for every file, make a deck
func main() {
	var pause = flag.Duration("loop", 0, "loop, pausing the specified duration between slides")
	var gridpct = flag.Float64("g", 10, "Grid percentage")
	var slidenum = flag.Int("slide", 0, "initial slide")
	flag.Parse()
	for _, f := range flag.Args() {
		dodeck(f, *pause, *slidenum, *gridpct)
	}
}