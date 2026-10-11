package game

import (
	"slices"

	"github.com/JMThomas00/Concord/sdk/arcade"
)

// The gunslingers: 12 x 14 pixels facing right (the outlaw is mirrored).
// H hat, h hat band, F face, B body, b belt, L legs, g gun, A drawn arm.
// A hat replaces the top three rows.
var (
	holster = []string{
		"....HHHH....", "..HHHHHHHH..", "....hhhh....", "....FFFF....", "....FFFF....", ".....FF.....",
		"...BBBBBB...", "..BBBBBBBB..", "..BB.BB.BB..", "..BB.bb.Bg..", "....BBBB....", "....L..L....", "....L..L....", "...LL..LL..."}
	drawn = []string{
		"....HHHH....", "..HHHHHHHH..", "....hhhh....", "....FFFF....", "....FFFF....", ".....FF.....",
		"...BBBBBBAAg", "..BBBBBBBB..", "..BB.BB.....", "..BB.bb.....", "....BBBB....", "....L..L....", "....L..L....", "...LL..LL..."}
)

// gunslinger draws one, 12 wide and 7 rows tall, from (x, row).
func gunslinger(c *arcade.Canvas, x, row int, hatID string, isDrawn, mirror bool, coat string, ghost bool) {
	h := HatByID(hatID)
	body := holster
	if isDrawn {
		body = drawn
	}
	rows := append(slices.Clone(h.Rows), body[3:]...)
	if mirror {
		rows = arcade.Flip(rows)
	}
	pal := map[rune]string{'F': "orange", 'B': coat, 'b': "tire", 'L': "tire", 'g': "comment", 'A': coat}
	for k, v := range h.Pal {
		pal[k] = v
	}
	if ghost {
		for k := range pal {
			pal[k] = "comment"
		}
	}
	c.Sprite(x, 2*row, rows, pal, 1)
}

// drawScene paints a backdrop over rows [top, top+h): sky, far hills and
// ground (its last three rows). frame moves snow and twinkles stars.
func drawScene(c *arcade.Canvas, sceneID string, x, top, w, h, frame int, grey bool) {
	s := SceneByID(sceneID)
	sky, ground, far := s.Sky, s.Ground, s.Far
	if grey {
		sky, ground, far = "line", "bg", "line"
	}
	c.Shade(x, top, w, h-3, sky)
	c.Shade(x, top+h-3, w, 3, ground)
	hills := []int{0, 0, 1, 1, 2, 2, 2, 1, 0, 0, 0, 1, 1, 1, 1, 0}
	for i := 0; i < w; i++ {
		for j := 0; j < hills[(x+i)%len(hills)]; j++ {
			c.Shade(x+i, top+h-4-j, 1, 1, far)
		}
	}
	if grey {
		return
	}
	sunX := x + w - 16
	switch {
	case s.Earth:
		c.Sprite(sunX, 2*(top+1), []string{"..####..", ".#cc###.", "##cc####", "###cc###", ".######.", "..####.."}, map[rune]string{'#': "cyan", 'c': "green"}, 1)
	case s.Moon:
		c.Sprite(sunX, 2*(top+1), []string{"..####..", ".#####..", "####....", "####....", ".#####..", "..####.."}, map[rune]string{'#': "fg"}, 1)
	case s.Sun != "":
		c.Sprite(sunX, 2*(top+1), []string{"..####..", ".######.", "########", "########", ".######.", "..####.."}, map[rune]string{'#': s.Sun}, 1)
	}
	if s.Stars {
		for k := 0; k < 14; k++ {
			ch := "."
			if (frame/10+k)%6 == 0 {
				ch = "·"
			}
			c.Text(x+(k*23+7)%w, top+1+(k*5)%(max(1, h-6)), ch, "dim", sky, false)
		}
	}
	if s.Snow {
		for k := 0; k < 18; k++ {
			c.Text(x+(k*17+frame/2)%w, top+1+(k*7+frame/3)%(max(1, h-4)), "*", "fg", sky, false)
		}
	}
	if s.Vines {
		for i := 2; i < w; i += 8 {
			c.Text(x+i, top+h-4, "❦", "green", far, false)
		}
	}
}

// tumbleweed rolls across row (its middle), wrapping round.
func tumbleweed(c *arcade.Canvas, x, w, row, frame int) {
	tx := x + (frame*2/3)%(w+10) - 5
	bounce := []int{0, 1, 1, 0}[frame/3%4]
	c.Sprite(tx, 2*row-bounce, []string{".oo.", "o..o", "o..o", ".oo."}, map[rune]string{'o': "orangeD"}, 1)
}

// target is what you shoot at in practice: a bullseye on a post.
func target(c *arcade.Canvas, x, row int) {
	c.Sprite(x, 2*row, []string{".rrrr.", "rwwwwr", "rwrrwr", "rwwwwr", ".rrrr.", "..tt..", "..tt..", "..tt..", "..tt..", "..tt.."},
		map[rune]string{'r': "red", 'w': "fg", 't': "orangeD"}, 1)
}
