package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"strings"
	"sync"
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
	sym  bool
}

func (p Piece) transform(m Matrix) Piece {
	var posi = make([]Pos, 0, len(p.pos))
	for _, pos := range p.pos {
		posi = append(posi, m.Transform(pos))
	}
	return Piece{p.name, posi, p.sym}
}

func (p Piece) allVersions() []Piece {
	var res []Piece
	for t := range tx {
		if p.sym && t >= 4 {
			return res
		}
		res = append(res, p.transform(tx[t]))
	}
	return res
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
	{0, 1},
	{-1, 0},
}

// Mirror mirrors a piece on its x axis
var Mirror = Matrix{
	{1, 0},
	{0, -1},
}

// tx contains all possible transformations.
var tx = []Matrix{
	Identity,
	Mirror,
	Rot90,
	Rot90.Mult(Mirror),
	Rot90.Mult(Rot90),
	Rot90.Mult(Rot90).Mult(Rot90),
	Rot90.Mult(Rot90).Mult(Mirror),
	Rot90.Mult(Rot90).Mult(Rot90).Mult(Mirror),
}

// Move descries the position of a piece on the board.
type Move struct {
	Piece     Piece
	Translate Pos
}

func (m Move) String() string {
	return fmt.Sprintf("%s at position (%v): %v", m.Piece.name, m.Translate, m.image())
}

func (m Move) image() []Pos {
	var res []Pos
	for _, p := range m.Piece.pos {
		res = append(res, p.translate(m.Translate))
	}
	return res
}

const (
	// DimX is the height of the playing board.
	DimX = 5
	// DimY is the width of the playing board.
	DimY = 11
)

var pieces = []Piece{
	{"blue", []Pos{{0, 0}, {0, 1}, {0, 2}, {1, 0}}, false},
	{"green", []Pos{{0, 0}, {1, 0}, {2, 0}, {1, 1}}, true},
	{"lightblue", []Pos{{0, 0}, {1, 0}, {2, 0}, {2, 1}, {2, 2}}, true},
	{"maroon", []Pos{{0, 0}, {0, 1}, {1, 1}, {1, 2}}, true},
	{"mint", []Pos{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}}, false},
	{"olive", []Pos{{0, 0}, {1, 0}, {2, 0}, {0, 1}, {2, 1}}, true},
	{"orange", []Pos{{0, 0}, {1, 0}, {1, 1}, {1, 2}, {2, 1}}, false},
	{"pink", []Pos{{0, 0}, {0, 1}, {0, 2}, {1, 2}, {1, 3}}, false},
	{"red", []Pos{{0, 0}, {0, 1}, {0, 2}, {0, 3}, {1, 0}}, false},
	{"turquoise", []Pos{{0, 0}, {0, 1}, {1, 0}}, false},
	{"violet", []Pos{{0, 0}, {1, 0}, {1, 1}, {2, 1}, {2, 2}}, true},
	{"yellow", []Pos{{0, 0}, {0, 1}, {0, 2}, {0, 3}, {1, 1}}, false},
}

// Game is a sequence of moves.
type Game struct {
	moves []Move
	cells [DimX][DimY]bool
	count int
}

func (g *Game) add(piece Piece, pos Pos) (bool, error) {
	if g.count+len(piece.pos) > DimX*DimY {
		return false, fmt.Errorf("board is already full")
	}
	var image [5]Pos
	for i, p := range piece.pos {
		var pi = p.translate(pos)
		if pi[0] < 0 || pi[0] >= DimX || pi[1] < 0 || pi[1] >= DimY {
			return false, nil
		}
		if g.cells[pi[0]][pi[1]] {
			return false, nil
		}
		image[i] = pi
	}
	g.moves = append(g.moves, Move{piece, pos})
	g.count += len(piece.pos)
	for i := range piece.pos {
		g.cells[image[i][0]][image[i][1]] = true
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
		var pi = p.translate(m.Translate)
		g.cells[pi[0]][pi[1]] = false
	}
	g.moves = g.moves[:len(g.moves)-1]
	return nil
}

var (
	board      = flag.String("board", "xxxxxxxxxxx,xxxxxxxxxxx,xxxxxxxxxxx,xxxxxxxxxxx,xxxxxxxxxxx", "The board (0 for empty, x for occupied)")
	available  = flag.String("pieces", "", "the available pieces")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
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
				res.cells[x][y] = true
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
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
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
	cache := precompute(ps)
	res := g.solveP(cache)
	for r := range res {
		fmt.Println("Solution found", r)
	}
}

func precompute(ps []Piece) [][]Piece {
	var res [][]Piece
	for _, piece := range ps {
		res = append(res, piece.allVersions())
	}
	return res
}

func (g Game) solveP(ps [][]Piece) <-chan []Move {
	if len(ps) == 0 {
		return nil
	}
	var res = make(chan []Move)

	var wg sync.WaitGroup
	for x := 0; x < DimX; x++ {
		for y := 0; y < DimY; y++ {
			for _, piece := range ps[len(ps)-1] {
				piece := piece
				x := x
				y := y
				g2 := &Game{
					cells: g.cells,
					count: g.count,
				}
				wg.Add(1)
				go func() {
					defer wg.Done()
					ok, err := g2.add(piece, Pos{x, y})
					if err != nil {
						panic(err)
					}
					if !ok {
						return
					}
					if err := g2.solve(ps[:len(ps)-1], res); err != nil {
						panic(err)
					}
				}()
			}
		}
	}
	go func() {
		wg.Wait()
		fmt.Println("all done")
		close(res)
	}()
	return res
}

func (g *Game) solve(ps [][]Piece, ch chan<- []Move) error {
	if len(ps) == 0 {
		if g.count != DimX*DimY {
			return fmt.Errorf("no pieces left, but board is not full")
		}
		var res = make([]Move, len(g.moves))
		copy(res, g.moves)
		ch <- res
		return nil
	}
	for x := 0; x < DimX; x++ {
		for y := 0; y < DimY; y++ {
			for _, piece := range ps[len(ps)-1] {
				ok, err := g.add(piece, Pos{x, y})
				if err != nil {
					return err
				}
				if !ok {
					continue
				}
				if err := g.solve(ps[:len(ps)-1], ch); err != nil {
					return err
				}
				if err := g.pop(); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
