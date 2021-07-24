package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

// Pos describes a position. We use coordinates starting at the top-left origin, with
// x going down and y going right (like mathematical matrix index notation).
type Pos [2]int

func (p Pos) translate(p2 Pos) Pos {
	return Pos{p[0] + p2[0], p[1] + p2[1]}
}

// Piece represents a piece.
type Piece struct {
	name string
	pos  []Pos
}

// Matrix represents a 2D transformation.
type Matrix [2][2]int

// Transform transforms the position given the matrix.
func (m Matrix) Transform(p Pos) Pos {
	return Pos{
		m[0][0]*p[0] + m[0][1]*p[1],
		m[1][0]*p[0] + m[1][1]*p[1],
	}
}

// Mult multiplies the given matrices.
func (m Matrix) Mult(m2 Matrix) Matrix {
	return Matrix{
		{m[0][0]*m2[0][0] + m[0][1]*m2[1][0], m[0][0]*m2[0][1] + m[0][1]*m2[1][1]},
		{m[1][0]*m2[0][0] + m[1][1]*m2[1][0], m[1][0]*m2[0][1] + m[1][1]*m2[1][1]},
	}
}

// Identity is the identity matrix.
var Identity = Matrix{
	{1, 0},
	{0, 1},
}

// Rot90 is a Rotation by 90 degrees.
var Rot90 = Matrix{
	{0, -1},
	{1, 0},
}

// Mirror mirrors a piece on its x axis
var Mirror = Matrix{
	{-1, 0},
	{0, -1},
}

// tx contains all possible transformations.
var tx = []Matrix{
	Identity,
	Rot90,
	Rot90.Mult(Rot90),
	Rot90.Mult(Rot90).Mult(Rot90),
	Mirror,
	Rot90.Mult(Mirror),
	Rot90.Mult(Rot90).Mult(Mirror),
	Rot90.Mult(Rot90).Mult(Rot90).Mult(Mirror),
}

// Move descries the position of a piece on the board.
type Move struct {
	Piece     Piece
	Transform int
	Translate Pos
}

const (
	DimX = 5
	DimY = 11
)

var pieces = []Piece{
	{"blue", []Pos{{0, 0}, {0, 1}, {0, 2}, {1, 1}}},
	{"green", []Pos{{0, 0}, {0, 1}, {0, 2}, {1, 1}}},
	{"lightblue", []Pos{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {2, 0}}},
	{"maroon", []Pos{{0, 0}, {0, 1}, {1, 1}, {1, 2}}},
	{"mint", []Pos{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}}},
	{"olive", []Pos{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 2}}},
	{"orange", []Pos{{0, 0}, {1, 0}, {1, 1}, {1, 2}, {2, 1}}},
	{"pink", []Pos{{0, 0}, {0, 1}, {0, 2}, {1, 2}, {1, 3}}},
	{"red", []Pos{{0, 0}, {0, 1}, {0, 2}, {0, 3}, {1, 1}}},
	{"turquoise", []Pos{{0, 0}, {0, 1}, {1, 0}}},
	{"violet", []Pos{{0, 0}, {1, 0}, {1, 1}, {2, 1}, {2, 1}}},
	{"yellow", []Pos{{0, 0}, {0, 1}, {0, 2}, {0, 3}, {1, 1}}},
}

// Game is a sequence of moves.
type Game struct {
	moves []Move
	cells [5][11]string
	count int
}

func (g *Game) add(piece Piece, transform int, pos Pos) (bool, error) {
	if transform < 0 || transform >= len(tx) {
		return false, fmt.Errorf("invalid transform: %d", transform)
	}
	if g.count+len(piece.pos) > DimX*DimY {
		return false, fmt.Errorf("board is already full")
	}
	var image []Pos
	for _, p := range piece.pos {
		var pi = tx[transform].Transform(p).translate(pos)
		if pi[0] < 0 || pi[0] >= len(g.cells) || pi[1] < 0 || pi[1] >= len(g.cells[pi[0]]) {
			return false, nil
		}
		if len(g.cells[pi[0]][pi[1]]) > 0 {
			return false, nil
		}
		image = append(image, pi)
	}
	g.moves = append(g.moves, Move{piece, transform, pos})
	g.count += len(image)
	for _, pi := range image {
		g.cells[pi[0]][pi[1]] = piece.name
	}
	return true, nil
}

func (g *Game) pop() error {
	if len(g.moves) == 0 {
		return errors.New("failed to pop from empty game")
	}
	var m = g.moves[len(g.moves)-1]
	g.count -= len(m.Piece.pos)
	for _, p := range m.Piece.pos {
		var pi = tx[m.Transform].Transform(p)
		g.cells[pi[0]][pi[1]] = ""
	}
	g.moves = g.moves[:len(g.moves)-1]
	return nil
}

var (
	board     = flag.String("board", "xxxxxxxxxxx,xxxxxxxxxxx,xxxxxxxxxxx,xxxxxxxxxxx,xxxxxxxxxxx", "The board (0 for empty, x for occupied)")
	available = flag.String("pieces", "", "the available pieces")
)

func parseBoard(b string) (*Game, error) {
	var rows = strings.Split(b, ",")
	if len(rows) != DimX {
		return nil, fmt.Errorf("board %q has an invalid number of rows, got %d, want %d", b, len(rows), DimX)
	}
	var res = new(Game)
	for x, row := range rows {
		if len(row) != DimY {
			return nil, fmt.Errorf("row %q has an invalid number of items, got %d, want %d", row, len(row), DimY)
		}
		for y, c := range row {
			if c == 'x' {
				res.cells[x][y] = "x"
				res.count++
			}
		}
	}
	return res, nil
}

func parseAvailable(a string) ([]Piece, error) {
	var (
		ps  = strings.Split(a, ",")
		res []Piece
	)
	if len(a) == 0 {
		return res, nil
	}
	for _, p := range ps {
		if piece, ok := getPiece(p); ok {
			res = append(res, piece)
		} else {
			return nil, fmt.Errorf("unknown piece: %s", p)
		}
	}
	return res, nil
}

func getPiece(name string) (Piece, bool) {
	for _, pc := range pieces {
		if pc.name == name {
			return pc, true
		}
	}
	return Piece{}, false
}

func main() {
	var (
		g   *Game
		err error
	)
	flag.Parse()
	g, err = parseBoard(*board)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	ps, err := parseAvailable(*available)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	ok, err := g.solve(ps)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if ok {
		fmt.Println("Found solution")
		fmt.Println(g.cells)
	} else {
		fmt.Println("No solution found")
	}
	os.Exit(0)
}

func (g *Game) solve(ps []Piece) (bool, error) {
	if len(ps) == 0 {
		return g.count == DimX*DimY, nil
	}
	var pos []Pos
	for x := 0; x < DimX; x++ {
		for y := 0; y < DimY; y++ {
			if len(g.cells[x][y]) == 0 {
				pos = append(pos, Pos{x, y})
			}
		}
	}
	for _, p := range pos {
		for transform := range tx {
			ok, err := g.add(ps[len(ps)-1], transform, p)
			if err != nil {
				return false, err
			}
			if ok {
				ok, err := g.solve(ps[:len(ps)-1])
				if ok || err != nil {
					return ok, err
				}
			}
		}
	}
	return false, nil
}
