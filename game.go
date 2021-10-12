package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"math/rand"
	"os"
	"path"
	"runtime/pprof"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"golang.org/x/image/colornames"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var bulletImage *ebiten.Image
var flashImage *ebiten.Image

var numberPrinter = message.NewPrinter(language.English)

var colorBlood = color.RGBA{102, 0, 0, 255}

const (
	gunshotVolume    = 0.2
	vampireDieVolume = 0.15
	batDieVolume     = 1.5
	playerHurtVolume = 0.4
	playerDieVolume  = 1.6
	munchVolume      = 0.8

	spawnVampire = 1000
	spawnGarlic  = 13

	garlicActiveTime = 7 * time.Second
)

var startButtons = []ebiten.StandardGamepadButton{
	ebiten.StandardGamepadButtonRightBottom,
	ebiten.StandardGamepadButtonRightRight,
	ebiten.StandardGamepadButtonRightLeft,
	ebiten.StandardGamepadButtonRightTop,
	ebiten.StandardGamepadButtonFrontTopLeft,
	ebiten.StandardGamepadButtonFrontTopRight,
	ebiten.StandardGamepadButtonFrontBottomLeft,
	ebiten.StandardGamepadButtonFrontBottomRight,
	ebiten.StandardGamepadButtonCenterLeft,
	ebiten.StandardGamepadButtonCenterRight,
	ebiten.StandardGamepadButtonLeftStick,
	ebiten.StandardGamepadButtonRightStick,
	ebiten.StandardGamepadButtonLeftBottom,
	ebiten.StandardGamepadButtonLeftRight,
	ebiten.StandardGamepadButtonLeftLeft,
	ebiten.StandardGamepadButtonLeftTop,
	ebiten.StandardGamepadButtonCenterCenter,
}

type projectile struct {
	x, y  float64
	angle float64
	speed float64
	color color.Color
}

// game is an isometric demo game.
type game struct {
	w, h  int
	level *Level

	player *gamePlayer

	gameStartTime time.Time

	gameOverTime time.Time

	camScale   float64
	camScaleTo float64

	mousePanX, mousePanY int

	projectiles []*projectile

	batSS *BatSpriteSheet

	ojasSS *PlayerSpriteSheet

	heartImg      *ebiten.Image
	vampireImage1 *ebiten.Image
	vampireImage2 *ebiten.Image
	vampireImage3 *ebiten.Image
	garlicImage   *ebiten.Image

	overlayImg *ebiten.Image
	op         *ebiten.DrawImageOptions

	audioContext *audio.Context
	nextSound    []int
	soundBuffer  [][]*audio.Player

	lastBatSound time.Time

	gamepadIDs    []ebiten.GamepadID
	gamepadIDsBuf []ebiten.GamepadID
	activeGamepad ebiten.GamepadID

	initialButtonReleased bool

	tick int

	godMode    bool
	debugMode  bool
	cpuProfile *os.File
}

const sampleRate = 44100

// NewGame returns a new isometric demo game.
func NewGame() (*game, error) {
	g := &game{
		camScale:   2,
		camScaleTo: 2,
		mousePanX:  math.MinInt32,
		mousePanY:  math.MinInt32,
		op:         &ebiten.DrawImageOptions{},

		soundBuffer:   make([][]*audio.Player, numSounds),
		nextSound:     make([]int, numSounds),
		activeGamepad: -1,
	}

	g.audioContext = audio.NewContext(sampleRate)

	ebiten.SetCursorShape(ebiten.CursorShapeCrosshair)

	err := g.loadAssets()
	if err != nil {
		return nil, err
	}

	g.player, err = NewPlayer()
	if err != nil {
		return nil, err
	}

	err = g.reset()
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (g *game) loadAssets() error {
	var err error
	// Load SpriteSheets.
	g.ojasSS, err = LoadPlayerSpriteSheet()
	if err != nil {
		return fmt.Errorf("failed to load embedded spritesheet: %s", err)
	}

	g.batSS, err = LoadBatSpriteSheet()
	if err != nil {
		return fmt.Errorf("failed to load embedded spritesheet: %s", err)
	}

	f, err := assetsFS.Open("assets/weapons/bullet.png")
	if err != nil {
		return err
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}

	bulletImage = ebiten.NewImageFromImage(img)

	f, err = assetsFS.Open("assets/weapons/flash.png")
	if err != nil {
		return err
	}
	img, _, err = image.Decode(f)
	if err != nil {
		return err
	}

	flashImage = ebiten.NewImageFromImage(img)

	g.soundBuffer[SoundGunshot] = make([]*audio.Player, 4)
	g.soundBuffer[SoundVampireDie1] = make([]*audio.Player, 4)
	g.soundBuffer[SoundVampireDie2] = make([]*audio.Player, 4)
	g.soundBuffer[SoundBat] = make([]*audio.Player, 4)
	g.soundBuffer[SoundPlayerHurt] = make([]*audio.Player, 4)
	g.soundBuffer[SoundPlayerDie] = make([]*audio.Player, 4)
	g.soundBuffer[SoundMunch] = make([]*audio.Player, 4)

	for i := 0; i < 4; i++ {
		stream, err := loadWav(g.audioContext, "assets/audio/gunshot.wav")
		if err != nil {
			return err
		}
		g.soundBuffer[SoundGunshot][i] = stream

		stream, err = loadWav(g.audioContext, "assets/audio/vampiredie1.wav")
		if err != nil {
			return err
		}
		g.soundBuffer[SoundVampireDie1][i] = stream

		stream, err = loadWav(g.audioContext, "assets/audio/vampiredie2.wav")
		if err != nil {
			return err
		}
		g.soundBuffer[SoundVampireDie2][i] = stream

		stream, err = loadWav(g.audioContext, "assets/audio/bat.wav")
		if err != nil {
			return err
		}
		g.soundBuffer[SoundBat][i] = stream

		stream, err = loadWav(g.audioContext, "assets/audio/playerhurt.wav")
		if err != nil {
			return err
		}
		g.soundBuffer[SoundPlayerHurt][i] = stream

		stream, err = loadWav(g.audioContext, "assets/audio/playerdie.wav")
		if err != nil {
			return err
		}
		g.soundBuffer[SoundPlayerDie][i] = stream

		stream, err = loadWav(g.audioContext, "assets/audio/munch.wav")
		if err != nil {
			return err
		}
		g.soundBuffer[SoundMunch][i] = stream
	}

	f, err = assetsFS.Open("assets/creeps/vampire1.png")
	if err != nil {
		return err
	}
	img, _, err = image.Decode(f)
	if err != nil {
		return err
	}
	g.vampireImage1 = ebiten.NewImageFromImage(img)

	f, err = assetsFS.Open("assets/creeps/vampire2.png")
	if err != nil {
		return err
	}
	img, _, err = image.Decode(f)
	if err != nil {
		return err
	}
	g.vampireImage2 = ebiten.NewImageFromImage(img)

	f, err = assetsFS.Open("assets/creeps/vampire3.png")
	if err != nil {
		return err
	}
	img, _, err = image.Decode(f)
	if err != nil {
		return err
	}
	g.vampireImage3 = ebiten.NewImageFromImage(img)

	f, err = assetsFS.Open("assets/items/garlic.png")
	if err != nil {
		return err
	}
	img, _, err = image.Decode(f)
	if err != nil {
		return err
	}

	g.garlicImage = ebiten.NewImageFromImage(img)

	f, err = assetsFS.Open("assets/ui/heart.png")
	if err != nil {
		return err
	}
	img, _, err = image.Decode(f)
	if err != nil {
		return err
	}

	g.heartImg = ebiten.NewImageFromImage(img)
	return nil
}

func (g *game) newItem(itemType int) *gameItem {
	sprite := g.garlicImage
	x, y := g.level.newSpawnLocation()
	return &gameItem{
		itemType: itemType,
		x:        x,
		y:        y,
		sprite:   sprite,
		level:    g.level,
		player:   g.player,
		health:   1,
	}
}

func (g *game) newCreep(creepType int) *gameCreep {
	sprites := []*ebiten.Image{
		g.vampireImage1,
		g.vampireImage2,
		g.vampireImage3,
		g.vampireImage2,
	}
	if creepType == TypeBat {
		sprites = []*ebiten.Image{
			g.batSS.Frame1,
			g.batSS.Frame2,
			g.batSS.Frame3,
			g.batSS.Frame4,
			g.batSS.Frame5,
			g.batSS.Frame6,
			g.batSS.Frame7,
		}
	}

	startingFrame := 0
	if len(sprites) > 1 {
		startingFrame = rand.Intn(len(sprites))
	}

	x, y := g.level.newSpawnLocation()
	return &gameCreep{
		creepType: creepType,
		x:         x,
		y:         y,
		sprites:   sprites,
		frames:    len(sprites),
		frame:     startingFrame,
		level:     g.level,
		player:    g.player,
		health:    1,
	}
}

func (g *game) reset() error {
	g.tick = 0

	var err error
	g.level, err = NewLevel()
	if err != nil {
		return fmt.Errorf("failed to create new level: %s", err)
	}
	g.level.player = g.player

	// Reset player score.
	g.player.score = 0

	// Reset player health.
	g.player.health = 3

	// Position player.
	g.player.x = float64(rand.Intn(108))
	g.player.y = float64(rand.Intn(108))

	// Remove projectiles.
	g.projectiles = nil

	// Spawn items.
	g.level.items = nil
	added := make(map[string]bool)
	for i := 0; i < spawnGarlic; i++ {
		itemType := itemTypeGarlic
		c := g.newItem(itemType)

		addedItem := fmt.Sprintf("%0.0f-%0.0f", c.x, c.y)
		if added[addedItem] {
			// Already added a gameItem here.
			i--
			continue
		}

		g.level.items = append(g.level.items, c)
		added[addedItem] = true
	}

	// Spawn creeps.
	g.level.creeps = make([]*gameCreep, 1000)
	for i := 0; i < spawnVampire; i++ {
		creepType := TypeVampire
		c := g.newCreep(creepType)

		addedCreep := fmt.Sprintf("%0.0f-%0.0f", c.x, c.y)
		if added[addedCreep] {
			// Already added a gameCreep here.
			i--
			continue
		}

		g.level.creeps[i] = c
		added[addedCreep] = true
	}
	return nil
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

// Update reads current user input and updates the game state.
func (g *game) Update() error {
	gamepadDeadZone := 0.1

	if ebiten.IsKeyPressed(ebiten.KeyEscape) || ebiten.IsWindowBeingClosed() {
		g.exit()
		return nil
	}

	if g.player.health <= 0 && !g.godMode {
		// Game over.
		if ebiten.IsKeyPressed(ebiten.KeyEnter) || (g.activeGamepad != -1 && ebiten.IsStandardGamepadButtonPressed(g.activeGamepad, ebiten.StandardGamepadButtonCenterRight)) {
			err := g.reset()
			if err != nil {
				return err
			}

			g.gameOverTime = time.Time{}
		}
		return nil
	}

	g.gamepadIDsBuf = inpututil.AppendJustConnectedGamepadIDs(g.gamepadIDsBuf[:0])
	for _, id := range g.gamepadIDsBuf {
		log.Printf("gamepad connected: %d", id)
		g.gamepadIDs = append(g.gamepadIDs, id)
	}
	for i, id := range g.gamepadIDs {
		if inpututil.IsGamepadJustDisconnected(id) {
			log.Printf("gamepad disconnected: %d", id)
			g.gamepadIDs = append(g.gamepadIDs[:i], g.gamepadIDs[i+1:]...)
		}

		if g.activeGamepad == -1 {
			for _, button := range startButtons {
				if ebiten.IsStandardGamepadButtonPressed(id, button) {
					log.Printf("gamepad activated: %d", id)
					g.activeGamepad = id
					ebiten.SetCursorMode(ebiten.CursorModeHidden)
					break
				}
			}
		}
	}

	if g.gameStartTime.IsZero() {
		var pressedKeys []ebiten.Key
		pressedKeys = inpututil.AppendPressedKeys(pressedKeys)
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) || g.activeGamepad != -1 || len(pressedKeys) > 0 {
			g.gameStartTime = time.Now()
		}
		return nil
	}

	biteThreshold := 0.75
	for _, c := range g.level.creeps {
		if c.health == 0 {
			continue
		}

		c.Update()

		cx, cy := c.Position()
		dx, dy := deltaXY(g.player.x, g.player.y, cx, cy)
		if dx <= biteThreshold && dy <= biteThreshold {
			if !g.godMode {
				g.player.health--
			}

			err := g.hurtCreep(c, -1)
			if err != nil {
				// TODO
				panic(err)
			}

			if g.player.health == 2 {
				g.playSound(SoundPlayerHurt, playerHurtVolume/2)
			} else if g.player.health == 1 {
				g.playSound(SoundPlayerHurt, playerHurtVolume)
			}

			g.addBloodSplatter(g.player.x, g.player.y)

			if g.player.health == 0 {
				ebiten.SetCursorShape(ebiten.CursorShapeDefault)

				g.gameOverTime = time.Now()

				// Play die sound.
				err := g.playSound(SoundPlayerDie, playerDieVolume)
				if err != nil {
					// TODO return err
					panic(err)
				}
			}
		} else if c.creepType == TypeBat && (dx <= 12 && dy <= 7) && rand.Intn(166) == 6 && time.Since(g.lastBatSound) >= 100*time.Millisecond {
			g.playSound(SoundBat, batDieVolume)
			g.lastBatSound = time.Now()
		}
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

	pan := 0.05

	// Pan camera.
	if g.activeGamepad != -1 {
		h := ebiten.StandardGamepadAxisValue(g.activeGamepad, ebiten.StandardGamepadAxisLeftStickHorizontal)
		v := ebiten.StandardGamepadAxisValue(g.activeGamepad, ebiten.StandardGamepadAxisLeftStickVertical)
		if v < -gamepadDeadZone || v > gamepadDeadZone || h < -gamepadDeadZone || h > gamepadDeadZone {
			g.player.x += h * pan
			g.player.y += v * pan
		}
	} else {
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
	}

	// Clamp camera position.
	g.player.x, g.player.y = g.level.Clamp(g.player.x, g.player.y)

	for _, item := range g.level.items {
		if item.health == 0 {
			continue
		}

		dx, dy := deltaXY(g.player.x, g.player.y, item.x, item.y)
		if dx <= 1 && dy <= 1 {
			item.health = 0
			g.playSound(SoundMunch, munchVolume)
			g.player.repelUntil = time.Now().Add(garlicActiveTime)
			g.player.score += item.useScore()
		}
	}

	fire := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)

	// Update player angle.
	if g.activeGamepad != -1 {
		h := ebiten.StandardGamepadAxisValue(g.activeGamepad, ebiten.StandardGamepadAxisRightStickHorizontal)
		v := ebiten.StandardGamepadAxisValue(g.activeGamepad, ebiten.StandardGamepadAxisRightStickVertical)
		if v < -gamepadDeadZone || v > gamepadDeadZone || h < -gamepadDeadZone || h > gamepadDeadZone {
			g.player.angle = angle(h, v, 0, 0)
			fire = true
		}
	} else {
		cx, cy := ebiten.CursorPosition()
		g.player.angle = angle(float64(cx), float64(cy), float64(g.w/2), float64(g.h/2))
	}

	if !g.initialButtonReleased {
		if fire {
			fire = false
		} else {
			g.initialButtonReleased = true
		}
	}

	// Update boolets.
	bulletHitThreshold := 0.5
	removed := 0
UPDATEPROJECTILES:
	for i, p := range g.projectiles {
		p.x += math.Cos(p.angle) * p.speed
		p.y += math.Sin(p.angle) * p.speed

		for _, c := range g.level.creeps {
			if c.health == 0 {
				continue
			}

			cx, cy := c.Position()
			dx, dy := deltaXY(p.x, p.y, cx, cy)
			if dx > bulletHitThreshold || dy > bulletHitThreshold {
				continue
			}

			err := g.hurtCreep(c, 1)
			if err != nil {
				return err
			}

			// Remove projectile
			g.projectiles = append(g.projectiles[:i-removed], g.projectiles[i-removed+1:]...)
			removed++

			continue UPDATEPROJECTILES
		}

		clampX, clampY := g.level.Clamp(p.x, p.y)
		if clampX != p.x || clampY != p.y {
			// Remove projectile
			g.projectiles = append(g.projectiles[:i-removed], g.projectiles[i-removed+1:]...)
			removed++
		}
	}

	// Fire boolets.
	if fire && time.Since(g.player.weapon.lastFire) >= g.player.weapon.cooldown {
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
		err := g.playSound(SoundGunshot, gunshotVolume)
		if err != nil {
			return err
		}
	}

	// Remove dead creeps.
	if g.tick%200 == 0 {
		removed = 0
		for i, creep := range g.level.creeps {
			if creep.health != 0 {
				continue
			}

			// Remove projectile
			g.level.creeps = append(g.level.creeps[:i-removed], g.level.creeps[i-removed+1:]...)
			removed++
		}
	}

	// Spawn garlic.
	if g.tick%144*20 == 0 {
		item := g.newItem(itemTypeGarlic)
		g.level.items = append(g.level.items, item)
	}

	// Spawn vampires.
	if g.tick > 144*5 && g.tick%288 == 0 {
		for i := 0; i < g.tick/1440; i++ {
			if rand.Intn(2) == 0 {
				continue
			}

			creepType := TypeVampire
			c := g.newCreep(creepType)

			g.level.creeps = append(g.level.creeps, c)
		}
	}

	// Spawn bats.
	if g.tick > 144*5 && g.tick%144 == 0 {
		for i := 0; i < g.tick/1440; i++ {
			if rand.Intn(6) == 0 {
				continue
			}

			creepType := TypeBat
			c := g.newCreep(creepType)

			g.level.creeps = append(g.level.creeps, c)
		}
	}

	// TODO debug only
	if inpututil.IsKeyJustPressed(ebiten.KeyV) {
		g.debugMode = !g.debugMode
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		g.godMode = !g.godMode
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		if g.cpuProfile == nil {
			log.Println("CPU profiling started...")

			homeDir, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			g.cpuProfile, err = os.Create(path.Join(homeDir, "cartillery.prof"))
			if err != nil {
				return err
			}
			if err := pprof.StartCPUProfile(g.cpuProfile); err != nil {
				return err
			}
		} else {
			log.Println("Profiling stopped")

			pprof.StopCPUProfile()
			g.cpuProfile.Close()
			g.cpuProfile = nil
		}
	}

	g.tick++
	return nil
}

func (g *game) drawText(target *ebiten.Image, y float64, scale float64, text string) {
	g.overlayImg.Clear()
	ebitenutil.DebugPrint(g.overlayImg, text)
	g.op.GeoM.Reset()
	g.op.GeoM.Scale(scale, scale)
	g.op.GeoM.Translate(float64(g.w/2)-(float64(len(text))*3*scale), y)
	target.DrawImage(g.overlayImg, g.op)
}

// Draw draws the game on the screen.
func (g *game) Draw(screen *ebiten.Image) {
	if g.gameStartTime.IsZero() {
		screen.Fill(colorBlood)

		g.drawText(screen, float64(g.h/2)-350, 16, "CAROTID")
		g.drawText(screen, float64(g.h/2)-100, 16, "ARTILLERY")

		g.drawText(screen, float64(g.h-210), 4, "KEYBOARD WASD")
		g.drawText(screen, float64(g.h-145), 4, "GAMEPAD RECOMMENDED")

		if time.Now().UnixMilli()%2000 < 1500 {
			g.drawText(screen, float64(g.h-80), 4, "PRESS ANY KEY OR BUTTON TO START")
		}

		return
	}

	gameOver := g.player.health <= 0 && !g.godMode

	var drawn int
	if !gameOver {
		drawn = g.renderLevel(screen)
	} else {
		// Game over.
		screen.Fill(colorBlood)

		g.drawText(screen, float64(g.h/2)-150, 16, "GAME OVER")

		if time.Since(g.gameOverTime).Milliseconds()%2000 < 1500 {
			g.drawText(screen, 8, 4, "PRESS ENTER OR START TO PLAY AGAIN")
		}
	}

	heartSpace := 64
	heartX := (g.w / 2) - ((heartSpace * g.player.health) / 2) + 16
	for i := 0; i < g.player.health; i++ {
		g.op.GeoM.Reset()
		g.op.GeoM.Translate(float64(heartX+(i*heartSpace)), 32)
		screen.DrawImage(g.heartImg, g.op)
	}

	scoreLabel := numberPrinter.Sprintf("%d", g.player.score)
	g.drawText(screen, float64(g.h-150), 8, scoreLabel)

	if g.godMode {
		// Draw God mode indicator.
		g.drawText(screen, float64(g.h-40), 2, " GOD")
	}

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

// tilePosition transforms X,Y coordinates into tile positions.
func (g *game) tilePosition(x, y float64) (float64, float64) {
	tileSize := float64(g.level.tileSize)
	return x * tileSize, y * tileSize
}

func (g *game) renderSprite(x float64, y float64, offsetx float64, offsety float64, angle float64, scale float64, alpha float64, sprite *ebiten.Image, target *ebiten.Image) int {
	x, y = g.tilePosition(x, y)

	// Skip drawing off-screen tiles.
	drawX, drawY := g.levelCoordinatesToScreen(x, y)
	padding := float64(g.level.tileSize) * 2
	if drawX+padding < 0 || drawY+padding < 0 || drawX > float64(g.w)+padding || drawY > float64(g.h)+padding {
		return 0
	}

	g.op.GeoM.Reset()

	g.op.GeoM.Scale(scale, scale)
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

	g.op.ColorM.Scale(1.0, 1.0, 1.0, alpha)

	target.DrawImage(sprite, g.op)

	g.op.ColorM.Reset()

	return 1
}

// renderLevel draws the current Level on the screen.
func (g *game) renderLevel(screen *ebiten.Image) int {
	var drawn int

	var t *Tile
	for y := 0; y < g.level.h; y++ {
		for x := 0; x < g.level.w; x++ {
			t = g.level.tiles[y][x]
			if t == nil {
				continue // No tile at this position.
			}

			for i := range t.sprites {
				drawn += g.renderSprite(float64(x), float64(y), 0, 0, 0, 1.0, 1.0, t.sprites[i], screen)
			}
		}
	}

	for _, item := range g.level.items {
		if item.health == 0 {
			continue
		}

		drawn += g.renderSprite(item.x, item.y, 0, 0, 0, 1.0, 1.0, item.sprite, screen)
	}

	for _, c := range g.level.creeps {
		if c.health == 0 {
			continue
		}

		drawn += g.renderSprite(c.x, c.y, 0, 0, 0, 1.0, 1.0, c.sprites[c.frame], screen)
		if c.frames > 1 && time.Since(c.lastFrame) >= 75*time.Millisecond {
			c.frame++
			if c.frame == c.frames {
				c.frame = 0
			}
			c.lastFrame = time.Now()
		}
	}

	for _, p := range g.projectiles {
		drawn += g.renderSprite(p.x, p.y, 0, 0, p.angle, 1.0, 1.0, bulletImage, screen)
	}

	repelTime := g.player.repelUntil.Sub(time.Now())
	if repelTime > 0 && repelTime < 7*time.Second {
		scale := repelTime.Seconds() + 1
		offset := 12 * scale
		alpha := 0.25
		if repelTime.Seconds() < 3 {
			alpha = repelTime.Seconds() / 12
		}
		drawn += g.renderSprite(g.player.x+0.25, g.player.y+0.25, -offset, -offset, 0, scale, alpha, g.garlicImage, screen)
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
	drawn += g.renderSprite(g.player.x, g.player.y, 0, 0, playerAngle, 1.0, 1.0, playerSprite, screen)
	if g.player.weapon != nil {
		drawn += g.renderSprite(g.player.x, g.player.y, 11*mul, 9, playerAngle, 1.0, 1.0, weaponSprite, screen)
	}

	flashDuration := 40 * time.Millisecond
	if time.Since(g.player.weapon.lastFire) < flashDuration {
		drawn += g.renderSprite(g.player.x, g.player.y, 39, -1, g.player.angle, 1.0, 1.0, flashImage, screen)
	}

	return drawn
}

func (g *game) playSound(sound int, volume float64) error {
	player := g.soundBuffer[sound][g.nextSound[sound]]
	g.nextSound[sound]++
	if g.nextSound[sound] > 3 {
		g.nextSound[sound] = 0
	}
	player.Pause()
	player.Rewind()
	player.SetVolume(volume)
	player.Play()
	return nil
}

func (g *game) hurtCreep(c *gameCreep, damage int) error {
	if damage == -1 {
		c.health = 0
		return nil
	}

	c.health -= damage
	if c.health > 0 {
		return nil
	}

	// Killed creep.
	g.player.score += c.killScore()

	// Play vampire die sound.

	var volume float64
	var dieSound int

	dieSound = SoundVampireDie1
	if rand.Intn(2) == 1 {
		dieSound = SoundVampireDie2
	}
	volume = vampireDieVolume
	/*
		if c.creepType == TypeBat {
			dieSound = SoundBat
			volume = batDieVolume
		} else {
			dieSound = SoundVampireDie1
			if rand.Intn(2) == 1 {
				dieSound = SoundVampireDie2
			}
			volume = vampireDieVolume
		}
	*/

	dx, dy := deltaXY(g.player.x, g.player.y, c.x, c.y)
	distance := dx
	if dy > dx {
		distance = dy
	}
	if distance > 9 {
		volume *= 0.7
	} else if distance > 6 {
		volume *= 0.85
	}

	err := g.playSound(dieSound, volume)
	if err != nil {
		return err
	}

	g.addBloodSplatter(c.x, c.y)

	return nil
}

func (g *game) levelCoordinatesToScreen(x, y float64) (float64, float64) {
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

	t := g.level.Tile(int(x), int(y))
	if t != nil {
		t.AddSprite(splatterSprite)
	}
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
