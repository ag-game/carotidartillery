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
	"sync"
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
	batVolume        = 1.0
	playerHurtVolume = 0.4
	playerDieVolume  = 1.6
	pickupVolume     = 0.8
	munchVolume      = 0.6

	spawnGarlic = 3

	garlicActiveTime = 7 * time.Second

	batSoundDelay = 250 * time.Millisecond

	screenPadding = 33

	startingHealth = 3
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
	x, y       float64
	angle      float64
	speed      float64
	color      color.Color
	colorScale float64
}

var blackSquare = ebiten.NewImage(32, 32)

// game is an isometric demo game.
type game struct {
	w, h  int
	level *Level

	levelNum int

	player *gamePlayer

	gameStartTime time.Time

	gameOverTime time.Time
	gameWon      bool

	winScreenBackground *ebiten.Image
	winScreenSun        *ebiten.Image
	winScreenSunY       float64
	winScreenColorScale float64

	camScale   float64
	camScaleTo float64

	mousePanX, mousePanY int

	projectiles []*projectile

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

	minLevelColorScale  float64
	minPlayerColorScale float64

	disableEsc bool

	godMode        bool
	noclipMode     bool
	muteAudio      bool
	debugMode      bool
	fullBrightMode bool
	cpuProfile     *os.File

	sync.Mutex
}

const sampleRate = 44100

// NewGame returns a new isometric demo game.
func NewGame() (*game, error) {
	g := &game{
		camScale:            2,
		camScaleTo:          2,
		mousePanX:           math.MinInt32,
		mousePanY:           math.MinInt32,
		activeGamepad:       -1,
		minLevelColorScale:  -1,
		minPlayerColorScale: -1,

		op: &ebiten.DrawImageOptions{},
	}

	g.audioContext = audio.NewContext(sampleRate)

	err := g.loadAssets()
	if err != nil {
		return nil, err
	}

	g.player, err = NewPlayer()
	if err != nil {
		return nil, err
	}

	blackSquare.Fill(color.Black)

	return g, nil
}

func (g *game) flashMessage(message string) {
	log.Println(message)

	g.flashMessageText = message
	g.flashMessageUntil = time.Now().Add(3 * time.Second)
}

func (g *game) loadAssets() error {
	var err error
	// Load SpriteSheets.
	ojasDungeonSS, err = LoadOjasDungeonSpriteSheet()
	if err != nil {
		return fmt.Errorf("failed to load embedded spritesheet: %s", err)
	}

	playerSS, err = LoadPlayerSpriteSheet()
	if err != nil {
		return fmt.Errorf("failed to load embedded spritesheet: %s", err)
	}

	batSS, err = LoadBatSpriteSheet()
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

func (g *game) nextLevel() error {
	g.player.soulsRescued = 0

	g.levelNum++
	if g.levelNum > 3 {
		g.showWinScreen()
		return nil
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
	g.level, err = NewLevel(g.levelNum, g.player)
	if err != nil {
		return fmt.Errorf("failed to create new level: %s", err)
	}

	// Position player.
	if g.levelNum > 1 {
		g.player.x, g.player.y = float64(g.level.enterX)+0.5, float64(g.level.enterY)-0.5
	} else {
		for {
			g.player.x, g.player.y = float64(rand.Intn(g.level.w)), float64(rand.Intn(g.level.h))
			if g.level.isFloor(g.player.x, g.player.y) {
				break
			}
		}
	}

	// Spawn items.
	g.level.items = nil
	for i := 0; i < spawnGarlic*g.levelNum; i++ {
		itemType := itemTypeGarlic
		c := g.newItem(itemType)
		g.level.items = append(g.level.items, c)
	}
	// Spawn starting garlic.
	item := g.newItem(itemTypeGarlic)
	for {
		garlicOffsetA := 8 - float64(rand.Intn(16))
		garlicOffsetB := 8 - float64(rand.Intn(16))
		startingGarlicX := g.player.x + 2 + garlicOffsetA
		startingGarlicY := g.player.y + 2 + garlicOffsetB

		if g.level.isFloor(startingGarlicX, startingGarlicY) {
			item.x = startingGarlicX
			item.y = startingGarlicY
			break
		}
	}
	g.level.items = append(g.level.items, item)

	// Spawn starting creeps.
	spawnAmount := 66
	if g.levelNum == 2 {
		spawnAmount = 133
	} else if g.levelNum == 3 {
		spawnAmount = 333
	}
	for i := 0; i < spawnAmount; i++ {
		g.level.addCreep(TypeVampire)
	}
	return nil
}

func (g *game) reset() error {
	log.Println("Starting a new game")

	g.tick = 0

	g.levelNum = 1

	g.gameStartTime = time.Now()

	g.gameOverTime = time.Time{}
	g.gameWon = false

	g.updateCursor()

	g.minLevelColorScale = -1
	g.minPlayerColorScale = -1

	g.player.hasTorch = true
	g.player.weapon = weaponUzi

	err := g.generateLevel()
	if err != nil {
		return err
	}

	// Reset player score.
	g.player.score = 0

	// Reset souls rescued.
	g.player.soulsRescued = 0

	// Reset player health.
	g.player.health = startingHealth

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
	if g.player.weapon != nil && g.player.weapon.spriteFlipped == nil {
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

func (g *game) updateCursor() {
	if g.activeGamepad != -1 || g.gameWon {
		ebiten.SetCursorMode(ebiten.CursorModeHidden)
		return
	}
	ebiten.SetCursorMode(ebiten.CursorModeVisible)
	ebiten.SetCursorShape(ebiten.CursorShapeCrosshair)
}

func (g *game) handlePlayerDeath() {
	if g.player.health > 0 {
		return
	}

	g.gameOverTime = time.Now()

	// Play die sound.
	err := g.playSound(SoundPlayerDie, playerDieVolume)
	if err != nil {
		// TODO return err
		panic(err)
	}

	g.updateCursor()
}

func (g *game) checkLevelComplete() {
	if g.player.soulsRescued < g.level.requiredSouls || !g.level.exitOpenTime.IsZero() {
		return
	}
	g.level.exitOpenTime = time.Now()

	// TODO preserve existing floor sprite

	t := g.level.tiles[g.level.exitY][g.level.exitX]
	t.sprites = nil
	t.AddSprite(sandstoneSS.FloorA)
	t.AddSprite(sandstoneSS.TopDoorOpenTL)

	t = g.level.tiles[g.level.exitY][g.level.exitX+1]
	t.sprites = nil
	t.AddSprite(sandstoneSS.FloorA)
	t.AddSprite(sandstoneSS.TopDoorOpenTR)

	t = g.level.tiles[g.level.exitY+1][g.level.exitX]
	t.sprites = nil
	t.AddSprite(sandstoneSS.FloorA)
	t.AddSprite(sandstoneSS.TopDoorOpenBL)

	t = g.level.tiles[g.level.exitY+1][g.level.exitX+1]
	t.sprites = nil
	t.AddSprite(sandstoneSS.FloorA)
	t.AddSprite(sandstoneSS.TopDoorOpenBR)

	for i := 1; i < 3; i++ {
		t = g.level.tiles[g.level.exitY-i][g.level.exitX]
		t.forceColorScale = 0

		t = g.level.tiles[g.level.exitY-i][g.level.exitX+1]
		t.forceColorScale = 0
	}

	// TODO add trigger entity or hardcode check
}

// Update reads current user input and updates the game state.
func (g *game) Update() error {
	g.Lock()
	defer g.Unlock()

	gamepadDeadZone := 0.1

	if (!g.disableEsc && ebiten.IsKeyPressed(ebiten.KeyEscape)) || ebiten.IsWindowBeingClosed() {
		g.exit()
		return nil
	}

	if !g.gameOverTime.IsZero() {
		if g.gameWon {
			return nil
		}
		// Game over.
		if ebiten.IsKeyPressed(ebiten.KeyEnter) || (g.activeGamepad != -1 && ebiten.IsStandardGamepadButtonPressed(g.activeGamepad, ebiten.StandardGamepadButtonCenterRight)) {
			err := g.reset()
			if err != nil {
				return err
			}
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
					g.updateCursor()
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

	liveCreeps := 0
	for _, c := range g.level.creeps {
		if c.health == 0 {
			continue
		}

		c.Update()

		if c.creepType == TypeTorch {
			continue
		}

		biteThreshold := 0.75
		if c.creepType == TypeSoul {
			biteThreshold = 0.25
		}

		// TODO can this move into creep?
		cx, cy := c.Position()
		dx, dy := deltaXY(g.player.x, g.player.y, cx, cy)
		if dx <= biteThreshold && dy <= biteThreshold {
			if c.creepType == TypeSoul {
				g.player.soulsRescued++
				g.player.score += 13
				err := g.hurtCreep(c, -1)
				if err != nil {
					// TODO
					panic(err)
				}
				g.checkLevelComplete()
			} else if !g.godMode && !c.repelled() {
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

				g.handlePlayerDeath()
			}
		} else if c.creepType == TypeBat && (dx <= 12 && dy <= 7) && rand.Intn(166) == 6 && time.Since(g.lastBatSound) >= batSoundDelay {
			g.playSound(SoundBat, batVolume)
			g.lastBatSound = time.Now()
		}

		if c.health > 0 {
			liveCreeps++
		}
	}
	g.level.liveCreeps = liveCreeps

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

	if g.noclipMode || g.level.isFloor(px, py) {
		g.player.x, g.player.y = px, py
	} else if g.level.isFloor(px, g.player.y) {
		g.player.x = px
	} else if g.level.isFloor(g.player.x, py) {
		g.player.y = py
	}

	for _, item := range g.level.items {
		if item.health == 0 {
			continue
		}

		dx, dy := deltaXY(g.player.x, g.player.y, item.x, item.y)
		if dx <= 1 && dy <= 1 {
			item.health = 0
			g.player.score += item.useScore() * g.levelNum

			if item.itemType == itemTypeGarlic {
				g.playSound(SoundMunch, munchVolume)
				g.player.garlicUntil = time.Now().Add(garlicActiveTime)
			} else if item.itemType == itemTypeHolyWater {
				g.playSound(SoundPickup, pickupVolume)
				g.player.health++
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
	bulletHitThreshold := 0.501
	bulletSeekThreshold := 2.0
	removed := 0
UPDATEPROJECTILES:
	for i, p := range g.projectiles {
		if p.speed == 0 {
			continue
		}
		speed := p.speed
		for {
			bx := p.x + math.Cos(p.angle)*speed
			by := p.y + math.Sin(p.angle)*speed

			if g.level.isFloor(bx, by) {
				p.x, p.y = bx, by
				break
			}

			speed *= .25
			if speed < .001 {
				// Remove projectile
				p.speed = 0
				p.colorScale = .01
				continue UPDATEPROJECTILES
			}
		}

		for _, c := range g.level.creeps {
			if c.health == 0 || c.creepType == TypeSoul {
				continue
			}

			cx, cy := c.Position()
			dx, dy := deltaXY(p.x, p.y, cx, cy)
			if dx > bulletHitThreshold || dy > bulletHitThreshold {
				if dx < bulletSeekThreshold && dy < bulletSeekThreshold {
					c.seekPlayer()
				}
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
	}

	// Fire boolets.
	if fire && g.player.weapon != nil && time.Since(g.player.weapon.lastFire) >= g.player.weapon.cooldown {
		p := &projectile{
			x:          g.player.x,
			y:          g.player.y,
			angle:      g.player.angle,
			speed:      0.35,
			color:      colornames.Yellow,
			colorScale: 1.0,
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
			if creep.health != 0 || creep.creepType == TypeTorch || creep.creepType == TypeSoul {
				continue
			}

			// Remove creep.
			g.level.creeps = append(g.level.creeps[:i-removed], g.level.creeps[i-removed+1:]...)
			removed++
		}
	}

	// Spawn garlic.
	if (g.tick > 0 && g.tick%(144*45) == 0) || rand.Intn(6666) == 0 {
		item := g.newItem(itemTypeGarlic)
		g.level.items = append(g.level.items, item)

	SPAWNGARLIC:
		for i := 0; i < 5; i++ {
			for _, levelItem := range g.level.items {
				if levelItem != item && item.itemType == itemTypeGarlic {
					dx, dy := deltaXY(item.x, item.y, levelItem.x, levelItem.y)
					if dx < 21 || dy < 21 {
						item.x, item.y = g.level.newSpawnLocation()
						continue SPAWNGARLIC
					}
				}
			}
			break
		}

		if g.debugMode {
			g.flashMessage("SPAWN GARLIC")
		}
	}

	// Spawn holy water.
	if g.tick%(144*30) == 0 || rand.Intn(6666) == 0 {
		item := g.newItem(itemTypeHolyWater)
		g.level.items = append(g.level.items, item)

	SPAWNHOLYWATER:
		for i := 0; i < 5; i++ {
			for _, levelItem := range g.level.items {
				if levelItem != item && item.itemType == itemTypeHolyWater {
					dx, dy := deltaXY(item.x, item.y, levelItem.x, levelItem.y)
					if dx < 21 || dy < 21 {
						item.x, item.y = g.level.newSpawnLocation()
						continue SPAWNHOLYWATER
					}
				}
			}
			break
		}

		if g.debugMode {
			g.flashMessage("SPAWN HOLY WATER")
		}
	}

	maxCreeps := 333
	if g.levelNum == 2 {
		maxCreeps = 666
	} else if g.levelNum == 3 {
		maxCreeps = 999
	}
	if len(g.level.creeps) < maxCreeps {
		// Spawn vampires.
		if g.tick%144 == 0 {
			spawnAmount := rand.Intn(1 + (g.tick / (144 * 9)))
			minCreeps := g.level.requiredSouls * 2
			if len(g.level.creeps) < minCreeps {
				spawnAmount *= 4
			}
			if g.debugMode && spawnAmount > 0 {
				g.flashMessage(fmt.Sprintf("SPAWN %d VAMPIRES", spawnAmount))
			}
			for i := 0; i < spawnAmount; i++ {
				g.level.addCreep(TypeVampire)
			}
		}

		// Spawn bats.
		if g.tick%(144*(4-g.levelNum)) == 0 {
			spawnAmount := g.tick / 288
			if spawnAmount < 1 {
				spawnAmount = 1
			} else if spawnAmount > 12 {
				spawnAmount = 12
			}
			spawnAmount = rand.Intn(spawnAmount)
			if g.debugMode && spawnAmount > 0 {
				g.flashMessage(fmt.Sprintf("SPAWN %d BATS", spawnAmount))
			}
			for i := 0; i < spawnAmount; i++ {
				g.level.addCreep(TypeBat)
			}
		}

		// Spawn ghosts.
		if false && g.tick%1872 == 0 { // Auto-spawn disabled.
			spawnAmount := g.tick / 1872
			if spawnAmount < 1 {
				spawnAmount = 1
			} else if spawnAmount > 6 {
				spawnAmount = 6
			}
			spawnAmount = rand.Intn(spawnAmount)
			if g.debugMode && spawnAmount > 0 {
				g.flashMessage(fmt.Sprintf("SPAWN %d GHOSTS", spawnAmount))
			}
			for i := 0; i < spawnAmount; i++ {
				g.level.addCreep(TypeGhost)
			}
		}
	}

	// Read user input.
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		g.muteAudio = !g.muteAudio
		if g.muteAudio {
			g.flashMessage("AUDIO MUTED")
		} else {
			g.flashMessage("AUDIO UNMUTED")
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyControl) {
		spawnAmount := 13
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyF):
			g.fullBrightMode = !g.fullBrightMode
			if g.fullBrightMode {
				g.flashMessage("FULLBRIGHT MODE ACTIVATED")
			} else {
				g.flashMessage("FULLBRIGHT MODE DEACTIVATED")
			}
		case inpututil.IsKeyJustPressed(ebiten.KeyG):
			g.godMode = !g.godMode
			if g.godMode {
				g.flashMessage("GOD MODE ACTIVATED")
			} else {
				g.flashMessage("GOD MODE DEACTIVATED")
			}
		case inpututil.IsKeyJustPressed(ebiten.KeyN):
			g.noclipMode = !g.noclipMode
			if g.noclipMode {
				g.flashMessage("NOCLIP MODE ACTIVATED")
			} else {
				g.flashMessage("NOCLIP MODE DEACTIVATED")
			}
		case inpututil.IsKeyJustPressed(ebiten.KeyV):
			g.debugMode = !g.debugMode
			if g.debugMode {
				g.flashMessage("DEBUG MODE ACTIVATED")
			} else {
				g.flashMessage("DEBUG MODE DEACTIVATED")
			}
		case inpututil.IsKeyJustPressed(ebiten.KeyP):
			if g.cpuProfile == nil {
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
				g.flashMessage("CPU PROFILING STOPPED")

				pprof.StopCPUProfile()
				g.cpuProfile.Close()
				g.cpuProfile = nil
			}
		case inpututil.IsKeyJustPressed(ebiten.Key1):
			for i := 0; i < spawnAmount; i++ {
				g.level.addCreep(TypeVampire)
			}
			g.flashMessage(fmt.Sprintf("SPAWNED %d VAMPIRES", spawnAmount))
		case inpututil.IsKeyJustPressed(ebiten.Key2):
			for i := 0; i < spawnAmount; i++ {
				g.level.addCreep(TypeBat)
			}
			g.flashMessage(fmt.Sprintf("SPAWNED %d BATS", spawnAmount))
		case inpututil.IsKeyJustPressed(ebiten.Key3):
			for i := 0; i < spawnAmount; i++ {
				g.level.addCreep(TypeGhost)
			}
			g.flashMessage(fmt.Sprintf("SPAWNED %d GHOSTS", spawnAmount))
		case inpututil.IsKeyJustPressed(ebiten.Key7):
			g.player.health++
			g.flashMessage("INCREASED HEALTH")
		case inpututil.IsKeyJustPressed(ebiten.Key8):
			// TODO Add garlic to inventory
			//g.flashMessage("+ GARLIC")
		case ebiten.IsKeyPressed(ebiten.KeyShift) && inpututil.IsKeyJustPressed(ebiten.KeyEqual):
			g.showWinScreen()
			g.flashMessage("WARPED TO WIN SCREEN")
		case inpututil.IsKeyJustPressed(ebiten.KeyMinus):
			if g.player.soulsRescued < g.level.requiredSouls {
				g.player.soulsRescued = g.level.requiredSouls
				g.checkLevelComplete()
				g.flashMessage("SKIPPED SOUL COLLECTION")
			} else {
				g.player.x, g.player.y = float64(g.level.exitX)+0.5, float64(g.level.exitY+2)
				g.flashMessage("WARPED TO EXIT")
			}
		case inpututil.IsKeyJustPressed(ebiten.KeyEqual):
			err := g.nextLevel()
			if err != nil {
				return err
			}
			g.flashMessage(fmt.Sprintf("WARPED TO LEVEL %d", g.levelNum))
		}
	}

	// Check if player is exiting level.
	if !g.level.exitOpenTime.IsZero() {
		exitThreshold := 1.1
		dx1, dy1 := deltaXY(g.player.x, g.player.y, float64(g.level.exitX), float64(g.level.exitY))
		dx2, dy2 := deltaXY(g.player.x, g.player.y, float64(g.level.exitX+1), float64(g.level.exitY))
		if (dx1 <= exitThreshold && dy1 <= exitThreshold) || (dx2 <= exitThreshold && dy2 <= exitThreshold) {
			err := g.nextLevel()
			if err != nil {
				return err
			}
		}
	}

	g.tick++
	return nil
}

func (g *game) drawText(target *ebiten.Image, x float64, y float64, scale float64, alpha float64, text string) {
	g.overlayImg.Clear()
	ebitenutil.DebugPrint(g.overlayImg, text)
	g.op.GeoM.Reset()
	g.op.GeoM.Scale(scale, scale)
	g.op.GeoM.Translate(x, y)
	g.op.ColorM.Scale(1, 1, 1, alpha)
	target.DrawImage(g.overlayImg, g.op)
	g.op.ColorM.Reset()
}

func (g *game) drawCenteredText(target *ebiten.Image, offsetX float64, y float64, scale float64, alpha float64, text string) {
	g.overlayImg.Clear()
	ebitenutil.DebugPrint(g.overlayImg, text)
	g.op.GeoM.Reset()
	g.op.GeoM.Scale(scale, scale)
	g.op.GeoM.Translate(float64(g.w/2)-(float64(len(text))*3*scale)+offsetX, y)
	g.op.ColorM.Scale(1, 1, 1, alpha)
	target.DrawImage(g.overlayImg, g.op)
	g.op.ColorM.Reset()
}

// Draw draws the game on the screen.
func (g *game) Draw(screen *ebiten.Image) {
	g.Lock()
	defer g.Unlock()

	if g.gameStartTime.IsZero() {
		screen.Fill(colorBlood)

		g.drawCenteredText(screen, 0, float64(g.h/2)-350, 16, 1.0, "CAROTID")
		g.drawCenteredText(screen, 0, float64(g.h/2)-100, 16, 1.0, "ARTILLERY")

		g.drawCenteredText(screen, 0, float64(g.h-210), 4, 1.0, "WASD + MOUSE = OK")
		g.drawCenteredText(screen, 0, float64(g.h-145), 4, 1.0, "FULLSCREEN + GAMEPAD = BEST")

		if time.Now().UnixMilli()%2000 < 1500 {
			g.drawCenteredText(screen, 0, float64(g.h-80), 4, 1.0, "PRESS ANY KEY OR BUTTON TO START")
		}

		return
	}

	var drawn int
	if g.gameOverTime.IsZero() || g.gameWon {
		if g.gameWon {
			g.drawProjectiles(screen)

			g.op.GeoM.Reset()
			g.op.ColorM.Reset()
			g.op.ColorM.Scale(1, 1, 1, g.winScreenColorScale)
			screen.DrawImage(g.winScreenBackground, g.op)

			g.op.GeoM.Reset()
			g.op.GeoM.Translate(float64(g.w)*0.75, g.winScreenSunY)
			g.op.ColorM.Reset()
			g.op.ColorM.Scale(g.winScreenColorScale, g.winScreenColorScale, g.winScreenColorScale, g.winScreenColorScale)
			screen.DrawImage(g.winScreenSun, g.op)
			g.op.ColorM.Reset()
		}
		drawn = g.renderLevel(screen)
	} else {
		drawn += g.drawProjectiles(screen)

		drawn += g.drawPlayer(screen)

		// Draw game over screen.
		img := ebiten.NewImage(g.w, g.h)
		img.Fill(colorBlood)

		a := g.minLevelColorScale
		if a == -1 {
			a = 1
		}

		g.op.GeoM.Reset()
		g.op.ColorM.Reset()
		g.op.ColorM.Scale(a, a, a, a)
		screen.DrawImage(img, g.op)
		g.op.ColorM.Reset()

		g.drawCenteredText(screen, 0, float64(g.h/2)-150, 16, a, "GAME OVER")

		if time.Since(g.gameOverTime).Milliseconds()%2000 < 1500 {
			g.drawCenteredText(screen, 0, 8, 4, a, "PRESS ENTER OR START TO PLAY AGAIN")
		}
	}

	if g.gameOverTime.IsZero() {
		// Draw health.
		healthScale := 1.5
		heartSpace := int(32 * healthScale)
		heartY := float64(g.h - screenPadding - heartSpace)
		for i := 0; i < g.player.health; i++ {
			g.op.GeoM.Reset()
			g.op.GeoM.Scale(healthScale, healthScale)
			g.op.GeoM.Translate(screenPadding+(float64((i)*heartSpace)), heartY)
			screen.DrawImage(imageAtlas[ImageHeart], g.op)
		}

		scale := 5.0
		soulsY := float64(g.h-int(scale*14)) - screenPadding
		if g.level.exitOpenTime.IsZero() {
			// Draw souls.
			soulsLabel := fmt.Sprintf("%d", g.level.requiredSouls-g.player.soulsRescued)

			soulImgSize := 46.0

			soulsX := float64(g.w-screenPadding) - (float64((len(soulsLabel)) * int(scale) * 6)) - 2 - soulImgSize

			soulImgScale := 1.5
			g.op.GeoM.Reset()
			g.op.GeoM.Translate((float64(g.w-screenPadding)-soulImgSize)/soulImgScale, (soulsY+19)/soulImgScale)
			g.op.GeoM.Scale(soulImgScale, soulImgScale)
			screen.DrawImage(ojasDungeonSS.Soul1, g.op)

			g.drawText(screen, soulsX, soulsY, scale, 1.0, soulsLabel)
		} else {
			// Draw exit message.
			if time.Since(g.level.exitOpenTime).Milliseconds()%2000 < 1500 {
				g.drawText(screen, float64(g.w-screenPadding)-(float64(9)*scale*6), soulsY, scale, 1.0, "EXIT OPEN")
			}
		}
	}

	flashTime := g.flashMessageUntil.Sub(time.Now())
	if flashTime > 0 {
		alpha := flashTime.Seconds() * 4
		if alpha > 1 {
			alpha = 1
		}
		g.drawCenteredText(screen, 0, float64(g.h-screenPadding-32), 2, alpha, g.flashMessageText)
	}

	if !g.gameOverTime.IsZero() && !g.gameWon {
		a := g.minLevelColorScale
		if a == -1 {
			a = 1
		}
		scale := 5
		scoreLabel := numberPrinter.Sprintf("%d", g.player.score)
		g.drawCenteredText(screen, 0, float64(g.h-(scale*14))-screenPadding, float64(scale), a, scoreLabel)
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
func (g *game) renderSprite(x float64, y float64, offsetx float64, offsety float64, angle float64, geoScale float64, colorScale float64, alpha float64, sprite *ebiten.Image, target *ebiten.Image) int {
	if g.minLevelColorScale != -1 && colorScale < g.minLevelColorScale {
		colorScale = g.minLevelColorScale
	}

	if alpha < .01 || colorScale < .01 {
		return 0
	}

	x, y = g.tilePosition(x, y)

	// Skip drawing off-screen tiles.
	drawX, drawY := g.levelCoordinatesToScreen(x, y)
	padding := float64(g.level.tileSize) * 2
	if drawX+padding < 0 || drawY+padding < 0 || drawX > float64(g.w)+padding || drawY > float64(g.h)+padding {
		return 0
	}

	g.op.GeoM.Reset()

	g.op.GeoM.Scale(geoScale, geoScale)
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
func (g *game) levelColorScale(x, y float64) float64 {
	if g.fullBrightMode {
		return 1
	}

	var v float64
	if g.player.hasTorch {
		v = colorScaleValue(x, y, g.player.x, g.player.y)
	}

	t := g.level.Tile(int(x), int(y))
	if t == nil {
		return 0
	}

	tileV := t.colorScale

	s := math.Min(1, v+tileV)

	if t.forceColorScale != 0 {
		return t.forceColorScale
	}

	return s
}

func (g *game) drawProjectiles(screen *ebiten.Image) int {
	var drawn int
	for _, p := range g.projectiles {
		colorScale := p.colorScale
		if colorScale == 1 {
			colorScale = g.levelColorScale(p.x, p.y)
		}

		alpha := 1.0
		if g.gameWon {
			//alpha = g.minLevelColorScale
		}
		// TODO if colorscale and gamewon, alpha is colorscale

		drawn += g.renderSprite(p.x, p.y, 0, 0, p.angle, 1.0, colorScale, alpha, imageAtlas[ImageBullet], screen)
	}
	return drawn
}

func (g *game) drawPlayer(screen *ebiten.Image) int {
	var drawn int

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

	var playerColorScale = g.levelColorScale(g.player.x, g.player.y)
	if g.minPlayerColorScale != -1 {
		playerColorScale = g.minPlayerColorScale
	}

	var weaponSprite *ebiten.Image

	playerSprite := playerSS.Frame1
	playerAngle := g.player.angle
	mul := float64(1)
	if g.player.weapon != nil {
		weaponSprite = g.player.weapon.spriteFlipped
	}
	if (g.player.angle > math.Pi/2 || g.player.angle < -1*math.Pi/2) && (g.gameOverTime.IsZero() || time.Since(g.gameOverTime) < 7*time.Second) {
		playerSprite = playerSS.Frame2
		playerAngle = playerAngle - math.Pi
		mul = -1
		if g.player.weapon != nil {
			weaponSprite = g.player.weapon.sprite
		}
	}
	drawn += g.renderSprite(g.player.x, g.player.y, 0, 0, playerAngle, 1.0, playerColorScale, 1.0, playerSprite, screen)
	if g.player.weapon != nil {
		drawn += g.renderSprite(g.player.x, g.player.y, 11*mul, 9, playerAngle, 1.0, playerColorScale, 1.0, weaponSprite, screen)
	}
	if g.player.hasTorch {
		drawn += g.renderSprite(g.player.x, g.player.y, -10*mul, 2, playerAngle, 1.0, playerColorScale, 1.0, sandstoneSS.TorchMulti, screen)
	}

	flashDuration := 40 * time.Millisecond
	if g.player.weapon != nil && time.Since(g.player.weapon.lastFire) < flashDuration {
		drawn += g.renderSprite(g.player.x, g.player.y, 39, -1, g.player.angle, 1.0, playerColorScale, 1.0, imageAtlas[ImageMuzzleFlash], screen)
	}

	return drawn
}

// renderLevel draws the current Level on the screen.
func (g *game) renderLevel(screen *ebiten.Image) int {
	var drawn int

	drawCreeps := func() {
		for _, c := range g.level.creeps {
			if c.health == 0 && c.creepType != TypeTorch {
				continue
			}

			a := 1.0
			if c.creepType == TypeSoul {
				a = 0.3
			}

			drawn += g.renderSprite(c.x, c.y, 0, 0, c.angle, 1.0, g.levelColorScale(c.x, c.y), a, c.sprites[c.frame], screen)
			if c.frames > 1 && time.Since(c.lastFrame) >= 75*time.Millisecond {
				c.frame++
				if c.frame == c.frames {
					c.frame = 0
				}
				c.lastFrame = time.Now()
			}
		}
	}

	// Render top tiles.
	var t *Tile
	for y := 0; y < g.level.h; y++ {
		for x := 0; x < g.level.w; x++ {
			t = g.level.tiles[y][x]
			if t == nil {
				continue // No tile at this position.
			}

			for i := range t.sprites {
				drawn += g.renderSprite(float64(x), float64(y), 0, 0, 0, 1.0, g.levelColorScale(float64(x), float64(y)), 1.0, t.sprites[i], screen)
			}
		}
	}

	for _, item := range g.level.items {
		if item.health == 0 {
			continue
		}

		drawn += g.renderSprite(item.x, item.y, 0, 0, 0, 1.0, g.levelColorScale(item.x, item.y), 1.0, item.sprite, screen)
	}

	if !g.gameWon {
		drawCreeps()
	}

	if !g.gameWon {
		drawn += g.drawProjectiles(screen)
	}

	drawn += g.drawPlayer(screen)

	if g.gameWon {
		drawCreeps()
	}

	// Render side and bottom walls a second time.
	if g.level.sideWalls != nil {
		for y := 0; y < g.level.h; y++ {
			for x := 0; x < g.level.w; x++ {
				t = g.level.sideWalls[y][x]
				if t == nil {
					continue // No tile at this position.
				}

				drawn += g.renderSprite(float64(x), float64(y), 0, 0, 0, 1.0, 1.0, 1.0, blackSquare, screen)
			}
		}
	}
	if g.level.otherWalls != nil {
		for y := 0; y < g.level.h; y++ {
			for x := 0; x < g.level.w; x++ {
				t = g.level.otherWalls[y][x]
				if t == nil {
					t = g.level.tiles[y][x]
					if t == nil || len(t.sprites) == 0 {
						drawn += g.renderSprite(float64(x), float64(y), 0, 0, 0, 1.0, 1.0, 1.0, blackSquare, screen)
					}
					continue // No tile at this position.
				}

				for i := range t.sprites {
					drawn += g.renderSprite(float64(x), float64(y), 0, 0, 0, 1.0, g.levelColorScale(float64(x), float64(y)), 1.0, t.sprites[i], screen)
				}
			}
		}
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
	if g.muteAudio {
		return nil
	}

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
	g.player.score += c.killScore() * g.levelNum

	if c.creepType == TypeTorch {
		// TODO play break sound
		c.frames = 1
		c.frame = 0
		c.sprites = []*ebiten.Image{
			sandstoneSS.TorchTop9,
		}
		g.level.bakePartialLightmap(int(c.x), int(c.y))
		return nil
	}

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
			volume = batVolume
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

	soul := g.level.addCreep(TypeSoul)
	soul.x, soul.y = c.x, c.y
	soul.moveX, soul.moveY = c.moveX/4, c.moveY/4
	soul.tick, soul.nextAction = c.tick, c.nextAction

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

func (g *game) showWinScreen() {
	if !g.gameOverTime.IsZero() {
		return
	}

	g.minLevelColorScale = 0.4

	g.gameWon = true
	g.gameOverTime = time.Now()

	g.updateCursor()

	g.player.health = 0
	g.player.garlicUntil = time.Time{}
	g.player.holyWaterUntil = time.Time{}

	g.level = newWinLevel(g.player)

	g.winScreenBackground = ebiten.NewImage(g.w, g.h)
	g.winScreenBackground.Fill(colornames.Deepskyblue)

	sunSize := 66
	g.winScreenSun = ebiten.NewImage(sunSize, sunSize)
	g.winScreenSun.Fill(color.RGBA{254, 231, 108, 255})

	g.winScreenSunY = float64(g.h/2) + float64(sunSize/2)

	go func() {
		p := g.player
		l := g.level

		var stars []*projectile

		addStar := func() {
			star := &projectile{
				x:          p.x + (0.5-rand.Float64())*66,
				y:          p.y + (0.5-rand.Float64())*66,
				colorScale: rand.Float64(),
			}
			g.projectiles = append(g.projectiles, star)
			stars = append(stars, star)
		}

		lastPlayerX := p.x
		updateStars := func() {
			if p.x == lastPlayerX {
				return
			}

			for _, star := range stars {
				star.x = p.x - (lastPlayerX - star.x)
			}
			lastPlayerX = p.x
		}

		// Add stars.
		numStars := 666
		for i := 0; i < numStars; i++ {
			addStar()
		}

		// Walk away.
		for i := 0; i < 36; i++ {
			p.x += 0.05
			updateStars()
			time.Sleep(time.Second / 144)
		}
		for i := 0; i < 288; i++ {
			p.x += 0.05 * (float64(288-i) / 288)
			updateStars()

			time.Sleep(time.Second / 144)
		}

		// Turn around.
		p.angle = math.Pi
		time.Sleep(time.Millisecond * 1750)

		// Throw weapon.
		weaponSprite := newCreep(TypeTorch, l, p)
		weaponSprite.x, weaponSprite.y = p.x, p.y
		weaponSprite.frames = 1
		weaponSprite.frame = 0
		weaponSprite.sprites = []*ebiten.Image{
			imageAtlas[ImageUzi],
		}

		p.weapon = nil
		l.creeps = append(l.creeps, weaponSprite)

		startX := 108

		doorX := float64(startX) - 0.4

		go func() {
			for i := 0; i < 144*2; i++ {
				if weaponSprite.x < doorX {
					for i, c := range l.creeps {
						if c == weaponSprite {
							l.creeps = append(l.creeps[:i], l.creeps[i+1:]...)
						}
					}
					return
				}

				weaponSprite.x -= 0.05
				if i < 100 {
					weaponSprite.y -= 0.005 * (float64(144-i) / 144)
				} else {
					weaponSprite.y += 0.01 * (float64(288-i) / 288)
				}
				weaponSprite.angle -= .1
				time.Sleep(time.Second / 144)
			}
		}()

		time.Sleep(time.Second / 2)

		// Throw torch.
		torchSprite := newCreep(TypeTorch, l, p)
		torchSprite.x, torchSprite.y = p.x, p.y
		torchSprite.frames = 1
		torchSprite.frame = 0
		torchSprite.sprites = []*ebiten.Image{
			sandstoneSS.TorchMulti,
		}

		p.hasTorch = false
		l.creeps = append(l.creeps, torchSprite)
		l.torches = append(l.torches, torchSprite)
		l.bakePartialLightmap(int(torchSprite.x), int(torchSprite.y))

		go func() {
			lastTorchX := torchSprite.x

			for i := 0; i < 144*3; i++ {
				if torchSprite.x < doorX {
					for i, c := range l.creeps {
						if c == torchSprite {
							l.creeps = append(l.creeps[:i], l.creeps[i+1:]...)
							l.torches = nil
						}
					}
				}

				torchSprite.x -= 0.05
				if i < 100 {
					torchSprite.y -= 0.005 * (float64(144-i) / 144)
				} else {
					torchSprite.y += 0.01 * (float64(288-i) / 288)
				}

				if lastTorchX-torchSprite.x >= 0.1 {
					l.bakePartialLightmap(int(torchSprite.x), int(torchSprite.y))
					lastTorchX = torchSprite.x
				}

				torchSprite.angle -= .1
				time.Sleep(time.Second / 144)
			}
		}()

		// Walk away.
		time.Sleep(time.Second)

		p.angle = 0
		for i := 0; i < 144; i++ {
			p.x += 0.05 * (float64(i) / 144)
			// Fade out stars.
			for _, star := range stars {
				star.colorScale -= 0.01
				if star.colorScale < 0 {
					star.colorScale = 0
				}
			}
			updateStars()
			time.Sleep(time.Second / 144)
		}

		var removedExistingStars bool

		for i := 0; i < 144*25; i++ {
			if p.health > 0 {
				// Game has restarted.
				return
			}

			if i > int(144*7) {
				if !removedExistingStars {
					// Remove existing stars.
					stars = nil
					g.projectiles = nil

					removedExistingStars = true
				}
			}

			if i > 144*12 {
				p.angle -= 0.0025 * (float64(i-(144*12)) / (144 * 3))

				for j := 0; j < 6; j++ {
					addStar()
				}
			}

			p.x += 0.05
			updateStars()

			if i > 144*11 {
				for _, star := range stars {
					pct := float64((144*15)-i) / 144 * 7

					star.x -= 0.1 * pct / 50

					// Apply warp effect.
					div := 100.0
					dx, dy := deltaXY(g.player.x, g.player.y, star.x, star.y)
					star.x, star.y = star.x-(dx/100)*pct/div-0.025, star.y+(dy/100)*pct/div-0.01
				}
			}

			time.Sleep(time.Second / 144)
		}
	}()

	go func() {
		// Animate sunrise.
		go func() {
			time.Sleep(5 * time.Second)
			for i := 0; i < 144*15; i++ {
				g.winScreenSunY -= 0.035

				if i > int(144*3.5) {
					g.minLevelColorScale += 0.0005
					if g.minLevelColorScale > 1 {
						g.minLevelColorScale = 1
					}
				}

				time.Sleep(time.Second / 144)
			}
		}()

		// Fade in sky.
		for i := 0.0001; i < 1; i *= 1.005 {
			g.winScreenColorScale = i

			time.Sleep(time.Second / 144)
		}

		time.Sleep(4 * time.Second)

		// Fade out win screen.

		for i := 0; i < 144*2; i++ {
			g.winScreenColorScale -= 0.005
			g.minLevelColorScale -= 0.005
			if g.minLevelColorScale > 0.6 {
				g.minPlayerColorScale = g.minLevelColorScale
			}
			time.Sleep(time.Second / 144)
		}

		g.Lock()
		for y := 0; y < g.level.h; y++ {
			for x := 0; x < g.level.w; x++ {
				g.level.tiles[y][x].sprites = nil
			}
		}
		g.Unlock()

		time.Sleep(7 * time.Second)

		// Fade in game over screen.

		g.gameWon = false

		defer func() {
			g.minLevelColorScale = -1
			g.minPlayerColorScale = -1
			g.updateCursor()
		}()

		for i := 0.01; i < 1; i *= 1.02 {
			if g.player.health > 0 {
				return
			}
			if i <= 1 {
				g.minLevelColorScale = i
			}
			time.Sleep(time.Second / 144)
		}
	}()
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
