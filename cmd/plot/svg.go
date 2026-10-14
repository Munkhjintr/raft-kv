package main

import (
	"fmt"
	"math"
	"os"
	"strings"
)

type series struct {
	name  string
	color string
	dash  bool
	vals  []float64
}

var xs = []int{1, 4, 16, 64}

func tick(v float64) string {
	if v >= 1000 {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%g", v)
}

func chart(file, title, ylabel string, loE, hiE int, ss []series) error {
	const w, h, l, r, t, b = 720, 440, 80, 20, 40, 110
	pw := float64(w - l - r)
	ph := float64(h - t - b)
	ly := func(v float64) float64 {
		f := (math.Log10(v) - float64(loE)) / float64(hiE-loE)
		return t + ph*(1-f)
	}
	lx := func(i int) float64 {
		return l + pw*(float64(i)+0.5)/float64(len(xs))
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" font-family="sans-serif" font-size="12">`, w, h, w, h)
	fmt.Fprintf(&sb, `<rect width="%d" height="%d" fill="white"/>`, w, h)
	fmt.Fprintf(&sb, `<text x="%d" y="24" font-size="16" font-weight="bold">%s</text>`, l, title)

	for e := loE; e <= hiE; e++ {
		v := math.Pow(10, float64(e))
		y := ly(v)
		fmt.Fprintf(&sb, `<line x1="%d" y1="%.1f" x2="%d" y2="%.1f" stroke="#ddd"/>`, l, y, w-r, y)
		fmt.Fprintf(&sb, `<text x="%d" y="%.1f" text-anchor="end">%s</text>`, l-8, y+4, tick(v))
	}
	for i, x := range xs {
		fmt.Fprintf(&sb, `<text x="%.1f" y="%d" text-anchor="middle">%d</text>`, lx(i), h-b+20, x)
	}
	fmt.Fprintf(&sb, `<text x="%.1f" y="%d" text-anchor="middle">concurrent clients</text>`, l+pw/2, h-b+40)
	fmt.Fprintf(&sb, `<text transform="translate(18,%.1f) rotate(-90)" text-anchor="middle">%s</text>`, t+ph/2, ylabel)

	for k, s := range ss {
		pts := make([]string, len(s.vals))
		for i, v := range s.vals {
			pts[i] = fmt.Sprintf("%.1f,%.1f", lx(i), ly(v))
		}
		dash := ""
		if s.dash {
			dash = ` stroke-dasharray="6 4"`
		}
		fmt.Fprintf(&sb, `<polyline points="%s" fill="none" stroke="%s" stroke-width="2.5"%s/>`, strings.Join(pts, " "), s.color, dash)
		for i, v := range s.vals {
			fmt.Fprintf(&sb, `<circle cx="%.1f" cy="%.1f" r="3.5" fill="%s"/>`, lx(i), ly(v), s.color)
		}
		x := l + (k%2)*320
		y := h - b + 66 + (k/2)*20
		fmt.Fprintf(&sb, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="2.5"%s/>`, x, y-4, x+28, y-4, s.color, dash)
		fmt.Fprintf(&sb, `<text x="%d" y="%d">%s</text>`, x+34, y, s.name)
	}
	sb.WriteString(`</svg>`)
	return os.WriteFile(file, []byte(sb.String()), 0o644)
}