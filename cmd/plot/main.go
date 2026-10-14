package main

import (
	"encoding/csv"
	"log"
	"os"
	"strconv"
)

type key [2]int

func load(path string) map[key][2]float64 {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	recs, err := csv.NewReader(f).ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	m := map[key][2]float64{}
	for _, r := range recs[1:] {
		n, _ := strconv.Atoi(r[0])
		c, _ := strconv.Atoi(r[1])
		o, _ := strconv.ParseFloat(r[3], 64)
		l, _ := strconv.ParseFloat(r[4], 64)
		m[key{n, c}] = [2]float64{o, l}
	}
	return m
}

func vals(m map[key][2]float64, nodes, idx int) []float64 {
	out := make([]float64, len(xs))
	for i, c := range xs {
		out[i] = m[key{nodes, c}][idx]
	}
	return out
}

func main() {
	before := load("bench-before.csv")
	after := load("bench-after.csv")
	if err := os.MkdirAll("docs", 0o755); err != nil {
		log.Fatal(err)
	}
	plots := []struct {
		file, title, ylabel string
		idx, lo, hi         int
	}{
		{"docs/throughput.svg", "Throughput: before vs after", "operations / second (log scale)", 0, 1, 6},
		{"docs/latency.svg", "Latency: before vs after", "latency, ms (log scale)", 1, -2, 2},
	}
	for _, p := range plots {
		ss := []series{
			{"3 nodes, before", "#2563eb", true, vals(before, 3, p.idx)},
			{"3 nodes, after", "#2563eb", false, vals(after, 3, p.idx)},
			{"5 nodes, before", "#ea580c", true, vals(before, 5, p.idx)},
			{"5 nodes, after", "#ea580c", false, vals(after, 5, p.idx)},
		}
		if err := chart(p.file, p.title, p.ylabel, p.lo, p.hi, ss); err != nil {
			log.Fatal(err)
		}
	}
}