package main

import "math/rand"

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Rect struct {
	X1 int `json:"x1"`
	Y1 int `json:"y1"`
	X2 int `json:"x2"`
	Y2 int `json:"y2"`
}

var dirs = []string{"up", "right", "down", "left"}

func makeRect(x1 int, y1 int, x2 int, y2 int) Rect {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	return Rect{x1, y1, x2, y2}
}

func randomRect(n int) Rect {
	x1 := rand.Intn(2*n) - n
	x2 := rand.Intn(2*n) - n
	y1 := rand.Intn(2*n) - n
	y2 := rand.Intn(2*n) - n
	return makeRect(x1, y1, x2, y2)
}

func (rect *Rect) Area() int {
	return (rect.X2 - rect.X1) * (rect.Y2 - rect.Y1)
}

func (rect *Rect) Perimeter() int {
	return ((rect.X2 - rect.X1) + (rect.Y2 - rect.Y1)) * 2
}

func (rect *Rect) Contains(x int, y int) bool {
	return x >= rect.X1 && x <= rect.X2 && y >= rect.Y1 && y <= rect.Y2
}

func (rect *Rect) Center() Point {
	return Point{
		(rect.X2 + rect.X1) / 2,
		(rect.Y2 + rect.Y1) / 2,
	}
}

func (rect *Rect) RandomPoint() Point {
	return Point{
		rect.X1 + rand.Intn(rect.X2-rect.X1+1),
		rect.Y1 + rand.Intn(rect.Y2-rect.Y1+1),
	}
}

func RandomDir() string {
	return dirs[rand.Intn(4)]
}
