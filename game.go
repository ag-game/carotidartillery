package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio/mp3"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"golang.org/x/image/colornames"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var spinner = []byte(`-\|/`)

var bulletImage *ebiten.Image
var flashImage *ebiten.Image

var numberPrinter = message.NewPrinter(language.English)

type projectile struct {
	x, y  float64
	angle float64
	speed float64
	color color.Color
}

// game is an isometric demo game.
type game struct {
	w, h         int
	currentLevel *Level

	soundGunshot     []byte
	soundVampireDie1 []byte
	soundVampireDie2 []byte
	soundGib         []byte
	soundDie         []byte

	player *gamePlayer

	gameOverTime time.Time

	camScale   float64
	camScaleTo float64

	mousePanX, mousePanY int

	spinnerIndex int

	creeps []*gameCreep

	projectiles []*projectile

	ojasSS *CharacterSpriteSheet

	overlayImg *ebiten.Image
	op         *ebiten.DrawImageOptions

	audioContext *audio.Context

	godMode   bool
	debugMode bool
}

const sampleRate = 48000

// NewGame returns a new isometric demo game.
func NewGame() (*game, error) {
	l, err := NewLevel()
	if err != nil {
		return nil, fmt.Errorf("failed to create new level: %s", err)
	}

	p, err := NewPlayer()
	if err != nil {
		return nil, err
	}

	g := &game{
		currentLevel: l,
		camScale:     2,
		camScaleTo:   2,
		mousePanX:    math.MinInt32,
		mousePanY:    math.MinInt32,
		player:       p,
		op:           &ebiten.DrawImageOptions{},
	}

	g.audioContext = audio.NewContext(sampleRate)

	g.player.x = float64(rand.Intn(108))
	g.player.y = float64(rand.Intn(108))

	// Load SpriteSheets.
	g.ojasSS, err = LoadCharacterSpriteSheet()
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded spritesheet: %s", err)
	}

	f, err := assetsFS.Open("assets/weapons/bullet.png")
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	bulletImage = ebiten.NewImageFromImage(img)

	f, err = assetsFS.Open("assets/weapons/flash.png")
	if err != nil {
		return nil, err
	}
	img, _, err = image.Decode(f)
	if err != nil {
		return nil, err
	}

	flashImage = ebiten.NewImageFromImage(img)

	loadSound := func(p string) ([]byte, error) {
		b, err := assetsFS.ReadFile(p)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		return b, nil
	}

	g.soundGunshot, err = loadSound("assets/audio/gunshot.mp3")
	if err != nil {
		return nil, err
	}

	g.soundGib, err = loadSound("assets/audio/gib.mp3")
	if err != nil {
		return nil, err
	}

	g.soundVampireDie1, err = loadSound("assets/audio/vampiredie1.mp3")
	if err != nil {
		return nil, err
	}

	g.soundVampireDie2, err = loadSound("assets/audio/vampiredie2.mp3")
	if err != nil {
		return nil, err
	}

	g.soundDie, err = loadSound("assets/audio/die.mp3")
	if err != nil {
		return nil, err
	}

	f, err = assetsFS.Open("assets/creeps/vampire.png")
	if err != nil {
		return nil, err
	}
	img, _, err = image.Decode(f)
	if err != nil {
		return nil, err
	}

	vampireImage := ebiten.NewImageFromImage(img)

	addedCreeps := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		c := NewCreep(vampireImage, g.currentLevel)

		safeSpace := 7.0
		dx, dy := deltaXY(g.player.x, g.player.y, c.x, c.y)
		if dx <= safeSpace || dy <= safeSpace {
			// Too close to the spawn point.
			continue
		}

		addedCreep := fmt.Sprintf("%0.0f-%0.0f", c.x, c.y)
		if addedCreeps[addedCreep] {
			// Already added a gameCreep here.
			continue
		}

		g.creeps = append(g.creeps, c)
		addedCreeps[addedCreep] = true
	}

	ebiten.SetCursorShape(ebiten.CursorShapeCrosshair)

	return g, nil
}

func (g *game) playSound(sound []byte, volume float64) error {
	stream, err := mp3.DecodeWithSampleRate(sampleRate, bytes.NewReader(sound))
	if err != nil {
		return err
	}

	player, err := g.audioContext.NewPlayer(stream)
	if err != nil {
		return err
	}
	player.SetVolume(volume)
	player.Play()
	return nil
}

// Update reads current user input and updates the game state.
func (g *game) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyEscape) || ebiten.IsWindowBeingClosed() {
		g.exit()
		return nil
	}

	if g.player.health <= 0 && !g.godMode {
		// Game over.
		return nil
	}

	for _, c := range g.creeps {
		c.Update()
	}

	// Update target zoom level.
	var scrollY float64
	if ebiten.IsKeyPressed(ebiten.KeyC) || ebiten.IsKeyPressed(ebiten.KeyPageDown) {
		scrollY = -0.25
	} else if ebiten.IsKeyPressed(ebiten.KeyE) || ebiten.IsKeyPressed(ebiten.KeyPageUp) {
		scrollY = .25
	} else {
		_, scrollY = ebiten.Wheel()
		if scrollY < -1 {
			scrollY = -1
		} else if scrollY > 1 {
			scrollY = 1
		}
	}
	g.camScaleTo += scrollY * (g.camScaleTo / 7)

	// Clamp target zoom level.
	if g.camScaleTo < 2 {
		g.camScaleTo = 2
	} else if g.camScaleTo > 4 {
		g.camScaleTo = 4
	}

	// Smooth zoom transition.
	div := 10.0
	if g.camScaleTo > g.camScale {
		g.camScale += (g.camScaleTo - g.camScale) / div
	} else if g.camScaleTo < g.camScale {
		g.camScale -= (g.camScale - g.camScaleTo) / div
	}

	// Pan camera via keyboard.
	pan := 0.05
	// TODO debug only
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		pan *= 5
	}

	if ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		g.player.x -= pan
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		g.player.x += pan
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
		g.player.y += pan
	}
	if ebiten.IsKeyPressed(ebiten.KeyUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
		g.player.y -= pan
	}

	// Clamp camera position.
	g.player.x, g.player.y = g.currentLevel.Clamp(g.player.x, g.player.y)

	// Update player angle.
	cx, cy := ebiten.CursorPosition()
	g.player.angle = math.Atan2(float64(cy-g.h/2), float64(cx-g.w/2))

	// Update boolets.
	bulletHitThreshold := 0.5
	removed := 0
	for i, p := range g.projectiles {
		p.x += math.Cos(p.angle) * p.speed
		p.y += math.Sin(p.angle) * p.speed

		for _, c := range g.creeps {
			if c.health == 0 {
				continue
			}

			cx, cy := c.Position()
			dx, dy := deltaXY(p.x, p.y, cx, cy)
			if dx <= bulletHitThreshold && dy <= bulletHitThreshold {
				c.health--

				// Killed creep.
				if c.health == 0 {
					g.player.score += c.killScore

					g.addBloodSplatter(cx, cy)

					// Play vampire die sound.
					dieSound := g.soundVampireDie1
					if rand.Intn(2) == 1 {
						dieSound = g.soundVampireDie2
					}
					err := g.playSound(dieSound, 0.25)
					if err != nil {
						return err
					}
				}

				// Remove projectile
				g.projectiles = append(g.projectiles[:i-removed], g.projectiles[i-removed+1:]...)
				removed++

				break
			}
		}
	}

	// Fire boolets.
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && time.Since(g.player.weapon.lastFire) >= g.player.weapon.cooldown {
		p := &projectile{
			x:     g.player.x,
			y:     g.player.y,
			angle: g.player.angle,
			speed: 0.35,
			color: colornames.Yellow,
		}
		g.projectiles = append(g.projectiles, p)

		g.player.weapon.lastFire = time.Now()

		// Play gunshot sound.
		err := g.playSound(g.soundGunshot, 0.4)
		if err != nil {
			return err
		}
	}

	// TODO debug only
	if inpututil.IsKeyJustPressed(ebiten.KeyV) {
		g.debugMode = !g.debugMode
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		g.godMode = !g.godMode
	}

	return nil
}

func (g *game) levelCoordinatesToScreen(x, y float64) (float64, float64) {
	px, py := g.tilePosition(g.player.x, g.player.y)
	py *= -1
	return ((x - px) * g.camScale) + float64(g.w/2.0), ((y + py) * g.camScale) + float64(g.h/2.0)
}

func (g *game) screenCoordinatesToLevel(x, y float64) (float64, float64) {
	// TODO reverse
	px, py := g.tilePosition(g.player.x, g.player.y)
	py *= -1
	return ((x - px) * g.camScale) + float64(g.w/2.0), ((y + py) * g.camScale) + float64(g.h/2.0)
}

func (g *game) addBloodSplatter(x, y float64) {
	splatterSprite := ebiten.NewImage(32, 32)

	for y := 8; y < 20; y++ {
		if rand.Intn(2) != 0 {
			continue
		}
		for x := 12; x < 20; x++ {
			if rand.Intn(5) != 0 {
				continue
			}
			splatterSprite.Set(x, y, colornames.Red)
		}
	}
	for y := 2; y < 26; y++ {
		if rand.Intn(5) != 0 {
			continue
		}
		for x := 2; x < 26; x++ {
			if rand.Intn(12) != 0 {
				continue
			}
			splatterSprite.Set(x, y, colornames.Red)
		}
	}

	t := g.currentLevel.Tile(int(x), int(y))
	if t != nil {
		t.AddSprite(splatterSprite)
	}
}

// Draw draws the game on the screen.
func (g *game) Draw(screen *ebiten.Image) {
	gameOver := g.player.health <= 0 && !g.godMode

	var drawn int
	if !gameOver {
		drawn = g.renderLevel(screen)
	} else {
		// Game over.
		screen.Fill(color.RGBA{102, 0, 0, 255})

		if time.Since(g.gameOverTime).Milliseconds()%2000 < 1500 {
			g.overlayImg.Clear()
			ebitenutil.DebugPrint(g.overlayImg, "GAME OVER")
			g.op.GeoM.Reset()
			g.op.GeoM.Translate(3, 0)
			g.op.GeoM.Scale(16, 16)
			g.op.GeoM.Translate(float64(g.w/2)-495, float64(g.h/2)-200)
			screen.DrawImage(g.overlayImg, g.op)
		}
	}

	scoreLabel := numberPrinter.Sprintf("%d", g.player.score)

	g.overlayImg.Clear()
	ebitenutil.DebugPrint(g.overlayImg, scoreLabel)
	g.op.GeoM.Reset()
	g.op.GeoM.Scale(8, 8)
	g.op.GeoM.Translate(float64(g.w/2)-float64(24*len(scoreLabel)), float64(g.h-150))
	screen.DrawImage(g.overlayImg, g.op)

	if !g.debugMode {
		return
	}

	// Print game info.
	g.overlayImg.Clear()
	ebitenutil.DebugPrint(g.overlayImg, fmt.Sprintf("SPR  %d\nTPS  %0.0f\nFPS  %0.0f", drawn, ebiten.CurrentTPS(), ebiten.CurrentFPS()))
	g.op.GeoM.Reset()
	g.op.GeoM.Translate(3, 0)
	g.op.GeoM.Scale(2, 2)
	screen.DrawImage(g.overlayImg, g.op)
}

// Layout is called when the game's layout changes.
func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	s := ebiten.DeviceScaleFactor()
	w, h := int(s*float64(outsideWidth)), int(s*float64(outsideHeight))
	if w != g.w || h != g.h {
		g.w, g.h = w, h

		debugBox := image.NewRGBA(image.Rect(0, 0, g.w, 200))
		g.overlayImg = ebiten.NewImageFromImage(debugBox)
	}
	if g.player.weapon.spriteFlipped == nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(32, 0)
		spriteFlipped := ebiten.NewImageFromImage(g.player.weapon.sprite)
		spriteFlipped.Clear()
		spriteFlipped.DrawImage(g.player.weapon.sprite, op)
		g.player.weapon.spriteFlipped = spriteFlipped
	}
	return g.w, g.h
}

// tilePosition transforms X,Y coordinates into tile positions.
func (g *game) tilePosition(x, y float64) (float64, float64) {
	tileSize := float64(g.currentLevel.tileSize)
	return x * tileSize, y * tileSize
}

func (g *game) renderSprite(x float64, y float64, offsetx float64, offsety float64, angle float64, sprite *ebiten.Image, target *ebiten.Image) int {
	x, y = g.tilePosition(x, y)

	// Skip drawing off-screen tiles.
	drawX, drawY := g.levelCoordinatesToScreen(x, y)
	padding := float64(g.currentLevel.tileSize) * 2
	if drawX+padding < 0 || drawY+padding < 0 || drawX > float64(g.w)+padding || drawY > float64(g.h)+padding {
		return 0
	}

	g.op.GeoM.Reset()
	// Rotate
	g.op.GeoM.Translate(-16+offsetx, -16+offsety)
	g.op.GeoM.Rotate(angle)
	// Move to current isometric position.
	g.op.GeoM.Translate(x, y)
	// Translate camera position.
	px, py := g.tilePosition(g.player.x, g.player.y)
	g.op.GeoM.Translate(-px, -py)
	// Zoom.
	g.op.GeoM.Scale(g.camScale, g.camScale)
	// Center.
	g.op.GeoM.Translate(float64(g.w/2.0), float64(g.h/2.0))

	target.DrawImage(sprite, g.op)
	return 1
}

// renderLevel draws the current Level on the screen.
func (g *game) renderLevel(screen *ebiten.Image) int {
	var drawn int

	var t *Tile
	for y := 0; y < g.currentLevel.h; y++ {
		for x := 0; x < g.currentLevel.w; x++ {
			t = g.currentLevel.tiles[y][x]
			if t == nil {
				continue // No tile at this position.
			}

			for i := range t.sprites {
				drawn += g.renderSprite(float64(x), float64(y), 0, 0, 0, t.sprites[i], screen)
			}
		}
	}

	biteThreshold := 0.75
	for _, c := range g.creeps {
		if c.health == 0 {
			continue
		}

		cx, cy := c.Position()
		dx, dy := deltaXY(g.player.x, g.player.y, cx, cy)
		if dx <= biteThreshold && dy <= biteThreshold {
			g.player.health--

			if g.player.health == 0 && !g.godMode {
				ebiten.SetCursorShape(ebiten.CursorShapeDefault)

				g.gameOverTime = time.Now()

				// Play die sound.
				err := g.playSound(g.soundDie, 1.6)
				if err != nil {
					// TODO return err
					panic(err)
				}
			}
		}

		drawn += g.renderSprite(c.x, c.y, 0, 0, 0, c.sprite, screen)
	}

	for _, p := range g.projectiles {
		drawn += g.renderSprite(p.x, p.y, 0, 0, p.angle, bulletImage, screen)
	}

	playerSprite := g.ojasSS.Frame1
	playerAngle := g.player.angle
	weaponSprite := g.player.weapon.spriteFlipped
	mul := float64(1)
	if g.player.angle > math.Pi/2 || g.player.angle < -1*math.Pi/2 {
		playerSprite = g.ojasSS.Frame2
		playerAngle = playerAngle - math.Pi
		weaponSprite = g.player.weapon.sprite
		mul = -1
	}
	drawn += g.renderSprite(g.player.x, g.player.y, 0, 0, playerAngle, playerSprite, screen)
	if g.player.weapon != nil {
		drawn += g.renderSprite(g.player.x, g.player.y, 11*mul, 9, playerAngle, weaponSprite, screen)
	}

	flashDuration := 40 * time.Millisecond
	if time.Since(g.player.weapon.lastFire) < flashDuration {
		drawn += g.renderSprite(g.player.x, g.player.y, 39, -1, g.player.angle, flashImage, screen)
	}

	return drawn
}

func (g *game) exit() {
	os.Exit(0)
}

func deltaXY(x1, y1, x2, y2 float64) (dx float64, dy float64) {
	dx, dy = x1-x2, y1-y2
	if dx < 0 {
		dx *= -1
	}
	if dy < 0 {
		dy *= -1
	}
	return dx, dy
}
