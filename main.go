package main

import (
	"errors"
	"flag"
	"image"
	"image/color"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	errMissingPaths  = errors.New("missing paths")
	errInvalidFormat = errors.New("invalid image format")
	errInvalidRate   = errors.New("invalid rounding rate")
	errInvalidCorner = errors.New("invalid corner value")
)

// Settable has a Set method to set the color for a point.
type Settable interface {
	Set(x, y int, c color.Color)
}

var empty = color.RGBA{255, 255, 255, 0}

func main() {
	opts := parseOptions()

	paths, err := parsePaths(flag.Args())
	if err != nil {
		log.Fatalln(err)
	}

	wg := new(sync.WaitGroup)
	for _, p := range paths {
		wg.Add(1)
		go process(p, opts, wg)
	}
	wg.Wait()
}

func parsePaths(paths []string) ([]string, error) {
	names := make([]string, 0)
	if len(paths) == 0 {
		return nil, errMissingPaths
	}
	for _, p := range paths {
		matches, err := filepath.Glob(p)
		if err != nil {
			return nil, err
		}
		names = append(names, matches...)
	}
	return names, nil
}

func process(path string, opts *option, wg *sync.WaitGroup) {
	defer wg.Done()

	f, err := os.Open(path)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	m, fm, err := decode(f)
	if err != nil {
		log.Fatalln(err)
	}
	convert(&m, opts)

	outPath := opts.output
	if opts.owrite {
		outPath = path
	}
	if outPath == "" {
		outPath = buildOutPath(path, opts.prefix, opts.suffix)
	}
	outF, err := os.Create(outPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer outF.Close()

	err = encode(fm, outF, m)
	if err != nil {
		log.Fatalln(err)
	}
}

func convert(m *image.Image, opts *option) {
	b := (*m).Bounds()
	w, h := b.Dx(), b.Dy()
	r := (float64(min(w, h)) / 2) * opts.rate

	sm, ok := (*m).(Settable)
	if !ok {
		// Check if image is YCbCr format.
		ym, ok := (*m).(*image.YCbCr)
		if !ok {
			log.Fatalln(errInvalidFormat)
		}
		*m = yCbCrToRGBA(ym)
		sm = (*m).(Settable)
	}
	// Parallelize?
	for y := 0.0; y <= r; y++ {
		l := math.Round(r - math.Sqrt(2*y*r-y*y))
		if opts.corner.topL {
			for x := 0; x <= int(l); x++ {
				sm.Set(x-1, int(y)-1, empty)
			}
		}
		if opts.corner.topR {
			for x := 0; x <= int(l); x++ {
				sm.Set(w-x, int(y)-1, empty)
			}
		}
		if opts.corner.bottomL {
			for x := 0; x <= int(l); x++ {
				sm.Set(x-1, h-int(y), empty)
			}
		}
		if opts.corner.bottomR {
			for x := 0; x <= int(l); x++ {
				sm.Set(w-x, h-int(y), empty)
			}
		}
	}
}

func buildOutPath(path, prefix, suffix string) string {
	ext := filepath.Ext(path)
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, ext)
	newName := prefix + name + suffix + ext
	return filepath.Join(filepath.Dir(path), newName)
}
