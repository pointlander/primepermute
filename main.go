// Copyright 2017 The PrimePermute Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"sort"

	"github.com/gonum/plot"
	"github.com/gonum/plot/plotter"
	"github.com/gonum/plot/vg"
)

type Lyndon struct {
	Words [][]uint8
}

func (l *Lyndon) Factor(s []uint8) {
	k, m, n, words, max := 0, 1, len(s), l.Words[:0], len(s)
	if max > 256 {
		max = 256 + (max-256)/2
	}
	if cap(words) < max {
		words = make([][]uint8, 0, max)
	}

	for {
		switch sk, sm := s[k], s[m]; true {
		case sk < sm:
			k, m = 0, m+1
			if m < n {
				continue
			}
		case sk == sm:
			k, m = k+1, m+1
			if m < n {
				continue
			}
			fallthrough
		case sk > sm:
			split := m - k
			k, m, s, words = 0, 1, s[split:], append(words, s[:split])
			n = len(s)
			if n > 1 {
				continue
			}
		}
		break
	}
	l.Words = append(words, s)
}

type rotation struct {
	int
	s []uint8
}

type Rotations []rotation

func (r Rotations) Len() int {
	return len(r)
}

func less(a, b rotation) bool {
	la, lb, ia, ib := len(a.s), len(b.s), a.int, b.int
	for {
		if x, y := a.s[ia], b.s[ib]; x != y {
			return x < y
		}
		ia, ib = ia+1, ib+1
		if ia == la {
			ia = 0
		}
		if ib == lb {
			ib = 0
		}
		if ia == a.int && ib == b.int {
			break
		}
	}
	return false
}

func (r Rotations) Less(i, j int) bool {
	return less(r[i], r[j])
}

func (r Rotations) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func merge(left, right, out Rotations) {
	for len(left) > 0 && len(right) > 0 {
		if less(left[0], right[0]) {
			out[0], left = left[0], left[1:]
		} else {
			out[0], right = right[0], right[1:]
		}
		out = out[1:]
	}
	copy(out, left)
	copy(out, right)
}

func psort(in Rotations, s chan<- bool) {
	if len(in) < 1024 {
		sort.Sort(in)
		s <- true
		return
	}

	l, r, split := make(chan bool), make(chan bool), len(in)/2
	left, right := in[:split], in[split:]
	go psort(left, l)
	go psort(right, r)
	_, _ = <-l, <-r
	out := make(Rotations, len(in))
	merge(left, right, out)
	copy(in, out)
	s <- true
}

func Coder(block []byte) {
	var lyndon Lyndon
	var rotations Rotations
	wait := make(chan bool)
	var buffer []uint8

	if cap(buffer) < len(block) {
		buffer = make([]uint8, len(block))
	} else {
		buffer = buffer[:len(block)]
	}
	copy(buffer, block)
	lyndon.Factor(buffer)

	/* rotate */
	if length := len(block); cap(rotations) < length {
		rotations = make(Rotations, length)
	} else {
		rotations = rotations[:length]
	}
	r := 0
	for _, word := range lyndon.Words {
		for i := range word {
			rotations[r], r = rotation{i, word}, r+1
		}
	}

	go psort(rotations, wait)
	<-wait

	/* output the last character of each rotation */
	for i, j := range rotations {
		if j.int == 0 {
			j.int = len(j.s)
		}
		block[i] = j.s[j.int-1]
	}
}

func Decoder(buffer []byte) {
	length := len(buffer)
	input, major, minor := make([]byte, length), [256]int{}, make([]int, length)
	for k, v := range buffer {
		input[k], minor[k], major[v] = v, major[v], major[v]+1
	}

	sum := 0
	for k, v := range major {
		major[k], sum = sum, sum+v
	}

	j := length - 1
	for k := range input {
		for minor[k] != -1 {
			buffer[j], j, k, minor[k] = input[k], j-1, major[input[k]]+minor[k], -1
		}
	}
}

func getBits(i int) (bits string) {
	for j := 0; j < 16; j++ {
		if i&1 == 1 {
			bits = "1" + bits
		} else {
			bits = "0" + bits
		}
		i >>= 1
	}
	bits = "0000" + bits
	return
}

const (
	PrimeCount16 = 6542
)

var (
	gaps   = [PrimeCount16]uint8{2, 3, 1}
	primes = map[uint32]bool{
		2: true,
		3: true,
		5: true,
	}
)

func primes16() {
	n, increment, primeCount, lastPrime := uint32(7), uint32(1), uint32(3), uint32(5)
	for primeCount < PrimeCount16 {
		prime, square, isPrime := uint32(3), uint32(0), true
		for p := uint32(2); p < primeCount; p++ {
			prime += 2 * uint32(gaps[p])
			square = prime * prime
			if square > n {
				break
			}
			if n%prime == 0 {
				isPrime = false
				break
			}
		}
		if isPrime {
			gaps[primeCount] = uint8((n - lastPrime) / 2)
			lastPrime = n
			primes[n] = true
			primeCount++
		}
		n += 2 << (increment & 1)
		increment++
	}
}

/*
static uint8_t gaps[ PRIME_COUNT16 ] = { 2, 3, 1 };
static void primes16() {
   uint32_t n = 7;
   uint32_t increment = 1;
   uint32_t primeCount;
   uint32_t lastPrime = 5;
   for ( primeCount = 3;
         primeCount < PRIME_COUNT16;
         n += 2 << ( increment++ & 1 ) ) {
      uint32_t prime = 3;
      uint32_t square;
      uint_fast16_t p;
      for ( p = 2; p < primeCount; p++ ) {
         prime += 2 * gaps[ p ];
         square = prime * prime;
         if ( !( n % prime ) || ( square > n ) ) {
            break;
         }
      }
      if ( ( square > n ) || ( p == primeCount ) ) {
         gaps[ primeCount ] = ( n - lastPrime ) / 2;
         lastPrime = n;
         primeCount++;
      }
   }
}
*/

func main() {
	primes16()
	test := []byte("SIX.MIXED.PIXIES.SIFT.SIXTY.PIXIE.DUST.BOXES")
	Coder(test)
	fmt.Println(string(test))
	Decoder(test)
	fmt.Println(string(test))
	v := make(plotter.Values, 65536)
	vp := make(plotter.Values, 0)
	vnp := make(plotter.Values, 0)
	for i := 0; i < 65536; i++ {
		bits, c := getBits(i), 0
		buffer := []byte(bits)
		Decoder(buffer)
		for bits != string(buffer) {
			Decoder(buffer)
			c++
		}
		v[i] = float64(c)
		if primes[uint32(i)] {
			vp = append(vp, float64(c))
		} else {
			vnp = append(vnp, float64(c))
		}
		fmt.Printf("%v %v\n", i, c)
	}

	p, err := plot.New()
	if err != nil {
		panic(err)
	}
	p.Title.Text = "Histogram"

	h, err := plotter.NewHist(v, 32)
	if err != nil {
		panic(err)
	}
	p.Add(h)

	if err := p.Save(8*vg.Inch, 8*vg.Inch, "hist.png"); err != nil {
		panic(err)
	}

	p, err = plot.New()
	if err != nil {
		panic(err)
	}
	p.Title.Text = "Primes Histogram"

	h, err = plotter.NewHist(vp, 32)
	if err != nil {
		panic(err)
	}
	p.Add(h)

	if err := p.Save(8*vg.Inch, 8*vg.Inch, "primes.png"); err != nil {
		panic(err)
	}

	p, err = plot.New()
	if err != nil {
		panic(err)
	}
	p.Title.Text = "Not Primes Histogram"

	h, err = plotter.NewHist(vnp, 32)
	if err != nil {
		panic(err)
	}
	p.Add(h)

	if err := p.Save(8*vg.Inch, 8*vg.Inch, "not_primes.png"); err != nil {
		panic(err)
	}
}
