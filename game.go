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

var numberPrinter = message.NewPrinter(language.English)

var colorBlood = color.RGBA{102, 0, 0, 255}

const (
	gunshotVolume    = 0.2
	vampireDieVolume = 0.15
	batDieVolume     = 1.5
	playerHurtVolume = 0.4
	playerDieVolume  = 1.6
	munchVolume      = 0.8

	spawnVampire = 777
	spawnGarlic  = 6

	garlicActiveTime    = 7 * time.Second
	holyWaterActiveTime = time.Second

	maxCreeps = 3333 // TODO optimize and raise

	batSoundDelay = 250 * time.Millisecond
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

	levelNum int

	player *gamePlayer

	requiredSouls int
	spawnedPortal bool

	gameStartTime time.Time

	gameOverTime time.Time

	camScale   float64
	camScaleTo float64

	mousePanX, mousePanY int

	projectiles []*projectile

	batSS *BatSpriteSheet

	ojasSS *PlayerSpriteSheet

	overlayImg *ebiten.Image
	op         *ebiten.DrawImageOptions

	audioContext *audio.Context

	lastBatSound time.Time

	gamepadIDs    []ebiten.GamepadID
	gamepadIDsBuf []ebiten.GamepadID
	activeGamepad ebiten.GamepadID

	initialButtonReleased bool

	tick int

	flashMessageText  string
	flashMessageUntil time.Time

	godMode        bool
	noclipMode     bool
	debugMode      bool
	fullBrightMode bool
	cpuProfile     *os.File
}

const sampleRate = 44100

// NewGame returns a new isometric demo game.
func NewGame() (*game, error) {
	g := &game{
		camScale:      2,
		camScaleTo:    2,
		mousePanX:     math.MinInt32,
		mousePanY:     math.MinInt32,
		activeGamepad: -1,

		op: &ebiten.DrawImageOptions{},
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

func (g *game) flashMessage(message string) {
	g.flashMessageText = message
	g.flashMessageUntil = time.Now().Add(3 * time.Second)
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

	soundAtlas = loadSoundAtlas(g.audioContext)

	return nil
}

func (g *game) newItem(itemType int) *gameItem {
	sprite := imageAtlas[ImageGarlic]
	if itemType == itemTypeHolyWater {
		sprite = imageAtlas[ImageHolyWater]
	}
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
		imageAtlas[ImageVampire1],
		imageAtlas[ImageVampire2],
		imageAtlas[ImageVampire3],
		imageAtlas[ImageVampire2],
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

func (g *game) nextLevel() error {
	g.levelNum++
	if g.levelNum > 13 {
		log.Fatal("YOU WIN")
	}
	return g.generateLevel()
}

func (g *game) generateLevel() error {
	// Remove projectiles.
	g.projectiles = nil

	// Remove creeps.
	if g.level != nil {
		g.level.creeps = nil
	}

	var err error
	g.level, err = NewLevel(g.levelNum)
	if err != nil {
		return fmt.Errorf("failed to create new level: %s", err)
	}
	g.level.player = g.player

	// Position player.
	for {
		g.player.x = float64(rand.Intn(g.level.w))
		g.player.y = float64(rand.Intn(g.level.h))

		if g.level.isFloor(g.player.x, g.player.y, false) {
			break
		}
	}

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
	// Spawn starting garlic.
	item := g.newItem(itemTypeGarlic)
	for {
		garlicOffsetA := 8 - float64(rand.Intn(16))
		garlicOffsetB := 8 - float64(rand.Intn(16))
		startingGarlicX := g.player.x + 2 + garlicOffsetA
		startingGarlicY := g.player.y + 2 + garlicOffsetB

		if g.level.isFloor(startingGarlicX, startingGarlicY, false) {
			item.x = startingGarlicX
			item.y = startingGarlicY
			break
		}
	}
	g.level.items = append(g.level.items, item)

	// Spawn creeps.
	g.level.creeps = make([]*gameCreep, spawnVampire)
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

func (g *game) reset() error {
	log.Println("Starting a new game")

	g.tick = 0

	g.levelNum = 1

	err := g.generateLevel()
	if err != nil {
		return err
	}

	// Reset player score.
	g.player.score = 0

	// Reset souls rescued.
	g.player.soulsRescued = 0

	// Reset player health.
	g.player.health = 3

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

	g.resetExpiredTimers()

	biteThreshold := 0.75
	liveCreeps := 0
	for _, c := range g.level.creeps {
		if c.health == 0 {
			continue
		}

		c.Update()

		cx, cy := c.Position()
		dx, dy := deltaXY(g.player.x, g.player.y, cx, cy)
		if dx <= biteThreshold && dy <= biteThreshold {
			if !g.godMode && !c.repelled() {
				if g.player.holyWaters > 0 {
					// TODO g.playSound(SoundItemUseHolyWater, useholyWaterVolume)
					g.player.holyWaterUntil = time.Now().Add(holyWaterActiveTime)
					g.player.holyWaters--
				} else {
					err := g.hurtCreep(c, -1)
					if err != nil {
						// TODO
						panic(err)
					}

					g.player.health--

					if g.player.health == 2 {
						g.playSound(SoundPlayerHurt, playerHurtVolume/2)
					} else if g.player.health == 1 {
						g.playSound(SoundPlayerHurt, playerHurtVolume)
					}

					g.addBloodSplatter(g.player.x, g.player.y)
				}
			}

			if g.player.health == 0 {
				ebiten.SetCursorShape(ebiten.CursorShapeDefault)

				g.player.holyWaters = 0

				g.gameOverTime = time.Now()

				// Play die sound.
				err := g.playSound(SoundPlayerDie, playerDieVolume)
				if err != nil {
					// TODO return err
					panic(err)
				}
			}
		} else if c.creepType == TypeBat && (dx <= 12 && dy <= 7) && rand.Intn(166) == 6 && time.Since(g.lastBatSound) >= batSoundDelay {
			g.playSound(SoundBat, batDieVolume)
			g.lastBatSound = time.Now()
		}

		if c.health > 0 {
			liveCreeps++
		}
	}
	g.level.liveCreeps = liveCreeps

	// Clamp target zoom level.
	/*if g.camScaleTo < 2 {
		g.camScaleTo = 2
	} else if g.camScaleTo > 4 {
		g.camScaleTo = 4
	} TODO */

	// Update target zoom level.
	if g.debugMode {
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
	px, py := g.player.x, g.player.y
	if g.activeGamepad != -1 {
		h := ebiten.StandardGamepadAxisValue(g.activeGamepad, ebiten.StandardGamepadAxisLeftStickHorizontal)
		v := ebiten.StandardGamepadAxisValue(g.activeGamepad, ebiten.StandardGamepadAxisLeftStickVertical)
		if v < -gamepadDeadZone || v > gamepadDeadZone || h < -gamepadDeadZone || h > gamepadDeadZone {
			px += h * pan
			py += v * pan
		}
	} else {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			pan /= 2
		}

		if ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
			px -= pan
		}
		if ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
			px += pan
		}
		if ebiten.IsKeyPressed(ebiten.KeyDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
			py += pan
		}
		if ebiten.IsKeyPressed(ebiten.KeyUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
			py -= pan
		}
	}

	if g.noclipMode || g.level.isFloor(px, py, false) {
		g.player.x, g.player.y = px, py
	} else if g.level.isFloor(px, g.player.y, false) {
		g.player.x = px
	} else if g.level.isFloor(g.player.x, py, false) {
		g.player.y = py
	}

	for _, item := range g.level.items {
		if item.health == 0 {
			continue
		}

		dx, dy := deltaXY(g.player.x, g.player.y, item.x, item.y)
		if dx <= 1 && dy <= 1 {
			item.health = 0
			g.player.score += item.useScore()

			if item.itemType == itemTypeGarlic {
				g.playSound(SoundMunch, munchVolume)
				g.player.garlicUntil = time.Now().Add(garlicActiveTime)
			} else if item.itemType == itemTypeHolyWater {
				// TODO g.playSound(SoundItemPickup, munchVolume)
				g.player.holyWaters++
			}
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

		if !g.level.isFloor(p.x, p.y, true) {
			// Remove projectile
			g.projectiles = append(g.projectiles[:i-removed], g.projectiles[i-removed+1:]...)
			removed++
			// TODO Add bullet hole
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

			// Remove creep.
			g.level.creeps = append(g.level.creeps[:i-removed], g.level.creeps[i-removed+1:]...)
			removed++
		}
	}

	// Spawn garlic.
	if g.tick%(144*45) == 0 {
		item := g.newItem(itemTypeGarlic)
		g.level.items = append(g.level.items, item)

		if g.debugMode {
			log.Println("SPAWN GARLIC")
		}
	}

	// Spawn holy water.
	if g.tick%(144*120) == 0 || rand.Intn(660) == 0 { // TODO
		item := g.newItem(itemTypeHolyWater)
		g.level.items = append(g.level.items, item)

		if g.debugMode {
			log.Println("SPAWN HOLY WATER")
		}
	}

	if len(g.level.creeps) < maxCreeps {
		// Spawn vampires.
		if g.tick%144 == 0 {
			spawnAmount := rand.Intn(26 + (g.tick / (144 * 3)))
			if len(g.level.creeps) < 500 {
				spawnAmount *= 4
			}
			if g.debugMode && spawnAmount > 0 {
				log.Printf("SPAWN %d VAMPIRES", spawnAmount)
			}
			for i := 0; i < spawnAmount; i++ {
				creepType := TypeVampire
				c := g.newCreep(creepType)

				g.level.creeps = append(g.level.creeps, c)
			}
		}

		// Spawn bats.
		if g.tick%144 == 0 {
			spawnAmount := g.tick / 288
			if spawnAmount < 1 {
				spawnAmount = 1
			} else if spawnAmount > 12 {
				spawnAmount = 12
			}
			spawnAmount = rand.Intn(spawnAmount)
			if g.debugMode && spawnAmount > 0 {
				log.Printf("SPAWN %d BATS", spawnAmount)
			}
			for i := 0; i < spawnAmount; i++ {
				creepType := TypeBat
				c := g.newCreep(creepType)

				g.level.creeps = append(g.level.creeps, c)
			}
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		g.godMode = !g.godMode
		if g.godMode {
			g.flashMessage("GOD MODE ACTIVATED")
		} else {
			g.flashMessage("GOD MODE DEACTIVATED")
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		g.noclipMode = !g.noclipMode
		if g.noclipMode {
			g.flashMessage("NOCLIP MODE ACTIVATED")
		} else {
			g.flashMessage("NOCLIP MODE DEACTIVATED")
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		g.player.holyWaters++
		g.flashMessage("+ HOLY WATER")
	}
	if ebiten.IsKeyPressed(ebiten.KeyControl) && inpututil.IsKeyJustPressed(ebiten.Key6) {
		g.fullBrightMode = !g.fullBrightMode
		if g.fullBrightMode {
			g.flashMessage("FULLBRIGHT MODE ACTIVATED")
		} else {
			g.flashMessage("FULLBRIGHT MODE DEACTIVATED")
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) {
		err := g.nextLevel()
		if err != nil {
			return err
		}
		g.flashMessage(fmt.Sprintf("WARPED TO LEVEL %d", g.levelNum))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyV) {
		g.debugMode = !g.debugMode
		if g.debugMode {
			g.flashMessage("DEBUG MODE ACTIVATED")
		} else {
			g.flashMessage("DEBUG MODE DEACTIVATED")
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		if g.cpuProfile == nil {
			log.Println("CPU profiling started...")
			g.flashMessage("CPU PROFILING STARTED")

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
			g.flashMessage("CPU PROFILING STOPPED")

			pprof.StopCPUProfile()
			g.cpuProfile.Close()
			g.cpuProfile = nil
		}
	}

	g.tick++
	return nil
}

func (g *game) drawText(target *ebiten.Image, y float64, scale float64, alpha float64, text string) {
	g.overlayImg.Clear()
	ebitenutil.DebugPrint(g.overlayImg, text)
	g.op.GeoM.Reset()
	g.op.GeoM.Scale(scale, scale)
	g.op.GeoM.Translate(float64(g.w/2)-(float64(len(text))*3*scale), y)
	g.op.ColorM.Scale(1, 1, 1, alpha)
	target.DrawImage(g.overlayImg, g.op)
	g.op.ColorM.Reset()
}

// Draw draws the game on the screen.
func (g *game) Draw(screen *ebiten.Image) {
	if g.gameStartTime.IsZero() {
		screen.Fill(colorBlood)

		g.drawText(screen, float64(g.h/2)-350, 16, 1.0, "CAROTID")
		g.drawText(screen, float64(g.h/2)-100, 16, 1.0, "ARTILLERY")

		g.drawText(screen, float64(g.h-210), 4, 1.0, "WASD + MOUSE = OK")
		g.drawText(screen, float64(g.h-145), 4, 1.0, "FULLSCREEN + GAMEPAD = BEST")

		if time.Now().UnixMilli()%2000 < 1500 {
			g.drawText(screen, float64(g.h-80), 4, 1.0, "PRESS ANY KEY OR BUTTON TO START")
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

		g.drawText(screen, float64(g.h/2)-150, 16, 1.0, "GAME OVER")

		if time.Since(g.gameOverTime).Milliseconds()%2000 < 1500 {
			g.drawText(screen, 8, 4, 1.0, "PRESS ENTER OR START TO PLAY AGAIN")
		}
	}

	heartSpace := 32
	heartX := (g.w / 2) - ((heartSpace * g.player.health) / 2) + 8
	for i := 0; i < g.player.health; i++ {
		g.op.GeoM.Reset()
		g.op.GeoM.Translate(float64(heartX+(i*heartSpace)), 32)
		screen.DrawImage(imageAtlas[ImageHeart], g.op)
	}

	holyWaterSpace := 16
	holyWaterX := (g.w / 2) - ((holyWaterSpace * g.player.holyWaters) / 2)
	for i := 0; i < g.player.holyWaters; i++ {
		g.op.GeoM.Reset()
		g.op.GeoM.Translate(float64(holyWaterX+(i*holyWaterSpace)), 76)
		screen.DrawImage(imageAtlas[ImageHolyWater], g.op)
	}

	scoreLabel := numberPrinter.Sprintf("%d", g.player.score)
	g.drawText(screen, float64(g.h-150), 8, 1.0, scoreLabel)

	flashTime := g.flashMessageUntil.Sub(time.Now())
	if flashTime > 0 {
		alpha := flashTime.Seconds() * 4
		if alpha > 1 {
			alpha = 1
		}
		g.drawText(screen, float64(g.h-40), 2, alpha, g.flashMessageText)
	}

	if !g.debugMode {
		return
	}

	// Print game info.
	g.overlayImg.Clear()
	ebitenutil.DebugPrint(g.overlayImg, fmt.Sprintf("CRP  %d\nSPR  %d\nTPS  %0.0f\nFPS  %0.0f", g.level.liveCreeps, drawn, ebiten.CurrentTPS(), ebiten.CurrentFPS()))
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

// renderSprite renders a sprite on the screen.
func (g *game) renderSprite(x float64, y float64, offsetx float64, offsety float64, angle float64, scale float64, colorScale float64, alpha float64, sprite *ebiten.Image, target *ebiten.Image) int {
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

	g.op.ColorM.Scale(colorScale, colorScale, colorScale, alpha)

	target.DrawImage(sprite, g.op)

	g.op.ColorM.Reset()

	return 1
}

// Calculate color scale to apply shadows.
func (g *game) colorScale(x, y float64) float64 {
	if g.fullBrightMode {
		return 1
	}

	dx, dy := deltaXY(x, y, g.player.x, g.player.y)

	sD := 7 / (dx + dy)
	if sD > 1 {
		sD = 1
	}

	sDB := sD
	if dx > 4 {
		sDB *= 0.6 / (dx / 4)
	}
	if dy > 4 {
		sDB *= 0.6 / (dy / 4)
	}

	sD = sD * 2 * sDB
	if sD > 1 {
		sD = 1
	}

	return sD
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
				drawn += g.renderSprite(float64(x), float64(y), 0, 0, 0, 1.0, g.colorScale(float64(x), float64(y)), 1.0, t.sprites[i], screen)
			}
		}
	}

	for _, item := range g.level.items {
		if item.health == 0 {
			continue
		}

		drawn += g.renderSprite(item.x, item.y, 0, 0, 0, 1.0, g.colorScale(item.x, item.y), 1.0, item.sprite, screen)
	}

	for _, c := range g.level.creeps {
		if c.health == 0 {
			continue
		}

		drawn += g.renderSprite(c.x, c.y, 0, 0, 0, 1.0, g.colorScale(c.x, c.y), 1.0, c.sprites[c.frame], screen)
		if c.frames > 1 && time.Since(c.lastFrame) >= 75*time.Millisecond {
			c.frame++
			if c.frame == c.frames {
				c.frame = 0
			}
			c.lastFrame = time.Now()
		}
	}

	for _, p := range g.projectiles {
		drawn += g.renderSprite(p.x, p.y, 0, 0, p.angle, 1.0, g.colorScale(p.x, p.y), 1.0, imageAtlas[ImageBullet], screen)
	}

	repelTime := g.player.garlicUntil.Sub(time.Now())
	if repelTime > 0 && repelTime < 7*time.Second {
		scale := repelTime.Seconds() + 1
		offset := 12 * scale
		alpha := 0.25
		if repelTime.Seconds() < 3 {
			alpha = repelTime.Seconds() / 12
		}
		drawn += g.renderSprite(g.player.x+0.25, g.player.y+0.25, -offset, -offset, 0, scale, 1.0, alpha, imageAtlas[ImageGarlic], screen)
	}

	holyWaterTime := g.player.holyWaterUntil.Sub(time.Now())
	if holyWaterTime > 0 && holyWaterTime < time.Second {
		scale := (holyWaterTime.Seconds() + 1) * 2
		offset := 16 * scale
		alpha := 0.25
		if holyWaterTime.Seconds() < 3 {
			alpha = holyWaterTime.Seconds() / 2
		}
		drawn += g.renderSprite(g.player.x+0.25, g.player.y+0.25, -offset, -offset, 0, scale, 1.0, alpha, imageAtlas[ImageHolyWater], screen)
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
	drawn += g.renderSprite(g.player.x, g.player.y, 0, 0, playerAngle, 1.0, 1.0, 1.0, playerSprite, screen)
	if g.player.weapon != nil {
		drawn += g.renderSprite(g.player.x, g.player.y, 11*mul, 9, playerAngle, 1.0, 1.0, 1.0, weaponSprite, screen)
	}

	flashDuration := 40 * time.Millisecond
	if time.Since(g.player.weapon.lastFire) < flashDuration {
		drawn += g.renderSprite(g.player.x, g.player.y, 39, -1, g.player.angle, 1.0, 1.0, 1.0, imageAtlas[ImageMuzzleFlash], screen)
	}

	return drawn
}

func (g *game) resetExpiredTimers() {
	if !g.player.garlicUntil.IsZero() && g.player.garlicUntil.Sub(time.Now()) <= 0 {
		g.player.garlicUntil = time.Time{}
	}
	if !g.player.holyWaterUntil.IsZero() && g.player.holyWaterUntil.Sub(time.Now()) <= 0 {
		g.player.holyWaterUntil = time.Time{}
	}
}

func (g *game) playSound(sound int, volume float64) error {
	player := soundAtlas[sound][nextSound[sound]]
	nextSound[sound]++
	if nextSound[sound] > 3 {
		nextSound[sound] = 0
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
