package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

const (
	ScreenWidth  = 640
	ScreenHeight = 360
	AudioRate    = 44100
	BossSpawnSec = 15.0 // Presentation demo mode: Zeus spawns at 15s

	PlayerSpawnX = ScreenWidth / 2
	PlayerSpawnY = ScreenHeight/2 + 50

	ShieldFrames    = 360 // Aegis: 6s immunity
	RapidFrames     = 420 // Owl Sight: 7s triple shot
	PowerUpLife     = 600 // pickups stay 10s
	PowerUpInterval = 360 // a new pickup every 6s
	MaxPowerUps     = 3
)

// Required Palette
var (
	ColorBg      = color.RGBA{R: 0x0B, G: 0x0F, B: 0x1A, A: 0xFF}
	ColorMarble  = color.RGBA{R: 0xED, G: 0xEB, B: 0xE6, A: 0xFF}
	ColorCyan    = color.RGBA{R: 0x3D, G: 0xF5, B: 0xFF, A: 0xFF}
	ColorGold    = color.RGBA{R: 0xFF, G: 0xC9, B: 0x4A, A: 0xFF}
	ColorMagenta = color.RGBA{R: 0xFF, G: 0x3D, B: 0x7F, A: 0xFF}
	ColorWhite   = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}

	// Derived shades (darker tones of the palette for depth)
	ColorShadow      = color.RGBA{R: 0, G: 0, B: 0, A: 90}
	ColorMarbleDark  = color.RGBA{R: 0xA8, G: 0xA4, B: 0x9C, A: 0xFF}
	ColorMagentaDark = color.RGBA{R: 0x8A, G: 0x1A, B: 0x45, A: 0xFF}
	ColorSteel       = color.RGBA{R: 0x1A, G: 0x22, B: 0x33, A: 0xFF}
)

type Scene int

const (
	SceneTitle Scene = iota
	ScenePlay
	SceneGameOver
	SceneWin
)

// Sound synthesizer (pure procedural PCM, zero files)
type SoundFX struct {
	ctx          *audio.Context
	shootPCM     []byte
	hitPCM       []byte
	enemyDeadPCM []byte
	thunderPCM   []byte
	winPCM       []byte
	pickupPCM    []byte
	shieldPCM    []byte
}

func newSoundFX() *SoundFX {
	ctx := audio.NewContext(AudioRate)
	s := &SoundFX{
		ctx:          ctx,
		shootPCM:     genTone(700, 250, 0.08, 0.22),
		hitPCM:       genNoise(180, 50, 0.22, 0.35),
		enemyDeadPCM: genTone(350, 120, 0.12, 0.28),
		thunderPCM:   genNoise(300, 30, 0.55, 0.45),
		winPCM:       genFanfare(),
		pickupPCM:    genTone(600, 1250, 0.13, 0.25),
		shieldPCM:    genTone(280, 950, 0.30, 0.25),
	}
	return s
}

func (s *SoundFX) play(pcm []byte) {
	if len(pcm) == 0 {
		return
	}
	p := s.ctx.NewPlayerFromBytes(pcm)
	p.Play()
}

func genTone(f1, f2 float64, dur float64, vol float64) []byte {
	samples := int(dur * float64(AudioRate))
	buf := new(bytes.Buffer)
	phase := 0.0
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(samples)
		freq := f1 + (f2-f1)*t
		phase += 2.0 * math.Pi * freq / float64(AudioRate)
		env := 1.0 - t
		val := math.Sin(phase) * env * vol
		s16 := int16(val * 32767)
		_ = binary.Write(buf, binary.LittleEndian, s16)
		_ = binary.Write(buf, binary.LittleEndian, s16)
	}
	raw := buf.Bytes()
	return raw[:len(raw)&^3]
}

func genNoise(f1, f2 float64, dur float64, vol float64) []byte {
	samples := int(dur * float64(AudioRate))
	buf := new(bytes.Buffer)
	phase := 0.0
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(samples)
		freq := f1 - (f1-f2)*t
		phase += 2.0 * math.Pi * freq / float64(AudioRate)
		saw := 2.0 * (phase/(2.0*math.Pi) - math.Floor(0.5+phase/(2.0*math.Pi)))
		noise := (float64((i*67)%100)/50.0 - 1.0) * 0.4
		env := math.Pow(1.0-t, 1.4)
		val := (saw*0.6 + noise*0.4) * env * vol
		s16 := int16(val * 32767)
		_ = binary.Write(buf, binary.LittleEndian, s16)
		_ = binary.Write(buf, binary.LittleEndian, s16)
	}
	raw := buf.Bytes()
	return raw[:len(raw)&^3]
}

func genFanfare() []byte {
	notes := []float64{523.25, 659.25, 783.99, 1046.50} // C5, E5, G5, C6
	durPerNote := 0.14
	samplesPerNote := int(durPerNote * float64(AudioRate))
	buf := new(bytes.Buffer)
	for _, freq := range notes {
		phase := 0.0
		for i := 0; i < samplesPerNote; i++ {
			t := float64(i) / float64(samplesPerNote)
			phase += 2.0 * math.Pi * freq / float64(AudioRate)
			env := math.Max(0, 1.0-t*0.7)
			val := math.Sin(phase) * env * 0.28
			s16 := int16(val * 32767)
			_ = binary.Write(buf, binary.LittleEndian, s16)
			_ = binary.Write(buf, binary.LittleEndian, s16)
		}
	}
	raw := buf.Bytes()
	return raw[:len(raw)&^3]
}

// Entities
type Player struct {
	X           float64
	Y           float64
	HP          int
	MaxHP       int
	Speed       float64
	Angle       float64
	InvulnTimer int
	ShootTimer  int
	Radius      float64
	ShieldTimer int // Aegis immunity
	RapidTimer  int // Owl Sight triple shot
}

type Bolt struct {
	X    float64
	Y    float64
	VX   float64
	VY   float64
	Life int
}

type EnemyKind int

const (
	EnemyNode     EnemyKind = iota // classic chaser
	EnemyHarpy                     // fast, weaving
	EnemyMinotaur                  // tanky, charges
)

type Enemy struct {
	Kind     EnemyKind
	X        float64
	Y        float64
	Radius   float64
	Speed    float64
	HP       int
	MaxHP    int
	HitFlash int
	Heading  float64
	Phase    float64
	Side     float64 // which way to slide around obstacles (+1/-1)

	ChargeTimer int
	ChargeCD    int
	ChargeX     float64
	ChargeY     float64
}

type ObstacleKind int

const (
	ObstacleColumn ObstacleKind = iota
	ObstacleRubble
	ObstacleServer
)

type Obstacle struct {
	X    float64
	Y    float64
	R    float64
	Kind ObstacleKind
	Seed float64
}

type FloorTile struct {
	X, Y  float32
	Alpha uint8
	Crack bool
}

type PowerUpKind int

const (
	PowerHeal   PowerUpKind = iota // Ambrosia: +1 HP
	PowerShield                    // Aegis: immunity
	PowerRapid                     // Owl Sight: triple shot
)

type PowerUp struct {
	X    float64
	Y    float64
	Kind PowerUpKind
	Life int
	Bob  float64
}

type FloatText struct {
	X, Y    float64
	Msg     string
	Life    int
	MaxLife int
	Color   color.RGBA
}

type WarningSpot struct {
	X       float64
	Y       float64
	Timer   int
	MaxTime int
	Radius  float64
	Struck  bool
}

type LightningStrike struct {
	X      float64
	Y      float64
	Life   int
	Jitter [6]float32
}

type BossZeus struct {
	X           float64
	Y           float64
	Radius      float64
	HP          int
	MaxHP       int
	Active      bool
	Angle       float64
	AttackTimer int
	HitFlash    int
	Enraged     bool
	MinionTimer int
}

type Particle struct {
	X       float64
	Y       float64
	VX      float64
	VY      float64
	Life    int
	MaxLife int
	Size    float32
	Color   color.RGBA
}

type Game struct {
	scene       Scene
	player      *Player
	bolts       []*Bolt
	enemies     []*Enemy
	warnings    []*WarningSpot
	lightnings  []*LightningStrike
	particles   []*Particle
	obstacles   []*Obstacle
	tiles       []FloorTile
	powerups    []*PowerUp
	texts       []*FloatText
	boss        *BossZeus
	sound       *SoundFX
	font        *text.GoXFace
	canvas      *ebiten.Image
	drawOp      *ebiten.DrawImageOptions
	whitePixel  *ebiten.Image
	vs          []ebiten.Vertex
	is          []uint16
	shakeTimer  int
	shakeMag    float64
	timerFrames int
	tick        int
	score       int
	spawnTimer  int
	powerTimer  int
	gridOffset  float64
}

func NewGame() *Game {
	white := ebiten.NewImage(3, 3)
	white.Fill(color.White)

	g := &Game{
		scene:      SceneTitle,
		sound:      newSoundFX(),
		font:       text.NewGoXFace(basicfont.Face7x13),
		canvas:     ebiten.NewImage(ScreenWidth, ScreenHeight),
		drawOp:     &ebiten.DrawImageOptions{},
		whitePixel: white.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image),
		bolts:      make([]*Bolt, 0, 64),
		enemies:    make([]*Enemy, 0, 64),
		warnings:   make([]*WarningSpot, 0, 16),
		lightnings: make([]*LightningStrike, 0, 16),
		particles:  make([]*Particle, 0, 256),
		powerups:   make([]*PowerUp, 0, 8),
		texts:      make([]*FloatText, 0, 16),
		player: &Player{
			X:      PlayerSpawnX,
			Y:      PlayerSpawnY,
			HP:     5,
			MaxHP:  5,
			Speed:  3.2,
			Radius: 10,
		},
		boss: &BossZeus{
			X:      ScreenWidth / 2,
			Y:      100,
			Radius: 34,
			HP:     40,
			MaxHP:  40,
		},
	}
	g.genTerrain()
	return g
}

func (g *Game) resetPlay() {
	g.scene = ScenePlay
	g.player.X = PlayerSpawnX
	g.player.Y = PlayerSpawnY
	g.player.HP = g.player.MaxHP
	g.player.InvulnTimer = 0
	g.player.ShootTimer = 0
	g.player.ShieldTimer = 0
	g.player.RapidTimer = 0
	g.shakeTimer = 0
	g.shakeMag = 0

	g.bolts = g.bolts[:0]
	g.enemies = g.enemies[:0]
	g.warnings = g.warnings[:0]
	g.lightnings = g.lightnings[:0]
	g.particles = g.particles[:0]
	g.powerups = g.powerups[:0]
	g.texts = g.texts[:0]

	g.boss.HP = 40
	g.boss.MaxHP = 40
	g.boss.Active = false
	g.boss.AttackTimer = 60
	g.boss.HitFlash = 0
	g.boss.Enraged = false
	g.boss.MinionTimer = 0

	g.timerFrames = 0
	g.score = 0
	g.spawnTimer = 40
	g.powerTimer = 180 // first pickup after 3s

	g.genTerrain()
}

// ---------- Terrain ----------

func (g *Game) genTerrain() {
	g.obstacles = g.obstacles[:0]
	for tries := 0; len(g.obstacles) < 7 && tries < 400; tries++ {
		kind := ObstacleKind(rand.Intn(3))
		r := 15.0 + rand.Float64()*5
		switch kind {
		case ObstacleRubble:
			r = 12 + rand.Float64()*3
		case ObstacleServer:
			r = 14
		}
		x := 50 + rand.Float64()*(ScreenWidth-100)
		y := 70 + rand.Float64()*(ScreenHeight-120)
		if math.Hypot(x-PlayerSpawnX, y-PlayerSpawnY) < 80 {
			continue
		}
		ok := true
		for _, o := range g.obstacles {
			if math.Hypot(x-o.X, y-o.Y) < r+o.R+45 {
				ok = false
				break
			}
		}
		if ok {
			g.obstacles = append(g.obstacles, &Obstacle{X: x, Y: y, R: r, Kind: kind, Seed: rand.Float64() * math.Pi * 2})
		}
	}

	g.tiles = g.tiles[:0]
	for i := 0; i < 26; i++ {
		gx := rand.Intn(ScreenWidth / 32)
		gy := 1 + rand.Intn(ScreenHeight/32-1)
		g.tiles = append(g.tiles, FloorTile{
			X:     float32(gx * 32),
			Y:     float32(gy * 32),
			Alpha: uint8(8 + rand.Intn(14)),
			Crack: rand.Float64() < 0.4,
		})
	}
}

// pushOut moves a circle out of any obstacle it overlaps. Returns whether it collided.
func (g *Game) pushOut(x, y, r float64) (float64, float64, bool) {
	hit := false
	for _, o := range g.obstacles {
		dx, dy := x-o.X, y-o.Y
		d := math.Hypot(dx, dy)
		minD := r + o.R
		if d < minD {
			hit = true
			if d < 0.001 {
				dx, dy, d = 1, 0, 1
			}
			x = o.X + dx/d*minD
			y = o.Y + dy/d*minD
		}
	}
	return x, y, hit
}

func (g *Game) insideObstacle(x, y, pad float64) bool {
	for _, o := range g.obstacles {
		if math.Hypot(x-o.X, y-o.Y) < o.R+pad {
			return true
		}
	}
	return false
}

func (g *Game) freeSpot(pad float64) (float64, float64) {
	for i := 0; i < 50; i++ {
		x := 40 + rand.Float64()*(ScreenWidth-80)
		y := 60 + rand.Float64()*(ScreenHeight-100)
		if !g.insideObstacle(x, y, pad+6) && math.Hypot(x-g.player.X, y-g.player.Y) > 50 {
			return x, y
		}
	}
	return PlayerSpawnX, PlayerSpawnY - 60
}

// ---------- Update ----------

func (g *Game) Update() error {
	g.tick++
	if g.shakeTimer > 0 {
		g.shakeTimer--
	}

	switch g.scene {
	case SceneTitle:
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.resetPlay()
		}
	case ScenePlay:
		g.updatePlay()
	case SceneGameOver, SceneWin:
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.resetPlay()
		}
	}

	// Particle physics
	aliveParts := g.particles[:0]
	for _, p := range g.particles {
		p.X += p.VX
		p.Y += p.VY
		p.Life--
		if p.Life > 0 {
			aliveParts = append(aliveParts, p)
		}
	}
	g.particles = aliveParts

	// Floating texts
	aliveTexts := g.texts[:0]
	for _, t := range g.texts {
		t.Y -= 0.5
		t.Life--
		if t.Life > 0 {
			aliveTexts = append(aliveTexts, t)
		}
	}
	g.texts = aliveTexts

	return nil
}

func (g *Game) playerImmune() bool {
	return g.player.InvulnTimer > 0 || g.player.ShieldTimer > 0
}

func (g *Game) updatePlay() {
	g.timerFrames++
	g.gridOffset = math.Mod(float64(g.timerFrames)*0.4, 32)

	// 1. Player Input (WASD)
	dx, dy := 0.0, 0.0
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		dy -= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		dy += 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		dx -= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		dx += 1
	}

	if dx != 0 && dy != 0 {
		norm := 1.0 / math.Sqrt(2)
		dx *= norm
		dy *= norm
	}

	g.player.X += dx * g.player.Speed
	g.player.Y += dy * g.player.Speed

	// Clamp arena bounds
	margin := g.player.Radius + 6
	if g.player.X < margin {
		g.player.X = margin
	}
	if g.player.X > ScreenWidth-margin {
		g.player.X = ScreenWidth - margin
	}
	if g.player.Y < margin+20 {
		g.player.Y = margin + 20
	}
	if g.player.Y > ScreenHeight-margin {
		g.player.Y = ScreenHeight - margin
	}
	// Terrain blocks the drone
	g.player.X, g.player.Y, _ = g.pushOut(g.player.X, g.player.Y, g.player.Radius)

	// Aim angle towards mouse
	mx, my := ebiten.CursorPosition()
	g.player.Angle = math.Atan2(float64(my)-g.player.Y, float64(mx)-g.player.X)

	// Shoot cyan bolts on Left Click (triple shot with Owl Sight)
	if g.player.ShootTimer > 0 {
		g.player.ShootTimer--
	}
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && g.player.ShootTimer == 0 {
		spreads := []float64{0}
		g.player.ShootTimer = 10
		if g.player.RapidTimer > 0 {
			spreads = []float64{-0.16, 0, 0.16}
			g.player.ShootTimer = 7
		}
		boltSpeed := 8.5
		for _, s := range spreads {
			a := g.player.Angle + s
			g.bolts = append(g.bolts, &Bolt{
				X:    g.player.X + math.Cos(a)*14,
				Y:    g.player.Y + math.Sin(a)*14,
				VX:   math.Cos(a) * boltSpeed,
				VY:   math.Sin(a) * boltSpeed,
				Life: 70,
			})
		}
		g.sound.play(g.sound.shootPCM)
	}

	if g.player.InvulnTimer > 0 {
		g.player.InvulnTimer--
	}
	if g.player.ShieldTimer > 0 {
		g.player.ShieldTimer--
	}
	if g.player.RapidTimer > 0 {
		g.player.RapidTimer--
	}

	// 2. Update Bolts (terrain absorbs them)
	aliveBolts := g.bolts[:0]
	for _, b := range g.bolts {
		b.X += b.VX
		b.Y += b.VY
		b.Life--
		if g.insideObstacle(b.X, b.Y, 2) {
			g.burst(b.X, b.Y, 4, ColorMarble)
			continue
		}
		if b.Life > 0 && b.X >= 0 && b.X <= ScreenWidth && b.Y >= 0 && b.Y <= ScreenHeight {
			aliveBolts = append(aliveBolts, b)
		}
	}
	g.bolts = aliveBolts

	// 3. Enemy Spawning (increases rate over BossSpawnSec)
	elapsedSec := float64(g.timerFrames) / 60.0
	if elapsedSec < BossSpawnSec {
		g.spawnTimer--
		if g.spawnTimer <= 0 {
			g.spawnEnemy(pickEnemyKind(elapsedSec))
			progress := elapsedSec / BossSpawnSec
			g.spawnTimer = int(45.0 - progress*30.0)
		}
	} else if !g.boss.Active {
		// BossSpawnSec reached: ZEUS.EXE awakens!
		g.boss.Active = true
		g.boss.X = ScreenWidth / 2
		g.boss.Y = 110
		g.boss.AttackTimer = 60
		g.shakeTimer = 18
		g.shakeMag = 5.5
		g.sound.play(g.sound.thunderPCM)
		g.burst(g.boss.X, g.boss.Y, 50, ColorGold)
	}

	// 3b. Power-up spawning
	g.powerTimer--
	if g.powerTimer <= 0 {
		g.powerTimer = PowerUpInterval
		if len(g.powerups) < MaxPowerUps {
			x, y := g.freeSpot(14)
			g.spawnPowerUp(x, y, randomPowerKind())
		}
	}

	// 4. Update Enemies
	aliveEnemies := g.enemies[:0]
	for _, e := range g.enemies {
		e.Phase += 0.15
		if e.HitFlash > 0 {
			e.HitFlash--
		}

		edx := g.player.X - e.X
		edy := g.player.Y - e.Y
		dist := math.Hypot(edx, edy)
		ux, uy := 0.0, 0.0
		if dist > 0.1 {
			ux, uy = edx/dist, edy/dist
		}
		speed := e.Speed

		switch e.Kind {
		case EnemyHarpy:
			// Weave side to side while closing in
			w := math.Sin(e.Phase*0.6) * 0.9
			ux, uy = ux-uy*w, uy+ux*w
			if n := math.Hypot(ux, uy); n > 0 {
				ux, uy = ux/n, uy/n
			}
		case EnemyMinotaur:
			if e.ChargeTimer > 0 {
				e.ChargeTimer--
				speed *= 2.8
				ux, uy = e.ChargeX, e.ChargeY
			} else {
				if e.ChargeCD > 0 {
					e.ChargeCD--
				}
				if dist < 140 && e.ChargeCD == 0 {
					e.ChargeTimer = 26
					e.ChargeCD = 150
					e.ChargeX, e.ChargeY = ux, uy
				}
			}
		}

		nx, ny := e.X+ux*speed, e.Y+uy*speed
		px, py, blocked := g.pushOut(nx, ny, e.Radius)
		if blocked {
			// Slide around the obstacle
			px += -uy * speed * 0.9 * e.Side
			py += ux * speed * 0.9 * e.Side
			px, py, _ = g.pushOut(px, py, e.Radius)
			if e.Kind == EnemyMinotaur {
				e.ChargeTimer = 0
			}
		}
		e.X, e.Y = px, py
		if ux != 0 || uy != 0 {
			e.Heading = math.Atan2(uy, ux)
		}

		// Bullet collision
		for _, b := range g.bolts {
			if b.Life > 0 && math.Hypot(b.X-e.X, b.Y-e.Y) < e.Radius+4 {
				b.Life = 0
				e.HP--
				e.HitFlash = 4
				g.burst(b.X, b.Y, 4, ColorWhite)
				if e.HP <= 0 {
					break
				}
			}
		}
		if e.HP <= 0 {
			g.killEnemy(e)
			continue
		}

		// Player collision
		if math.Hypot(g.player.X-e.X, g.player.Y-e.Y) < e.Radius+g.player.Radius {
			if g.player.ShieldTimer > 0 {
				g.killEnemy(e) // the Aegis shatters attackers
				continue
			}
			if g.player.InvulnTimer == 0 {
				g.playerTakeDamage()
				g.burst(e.X, e.Y, 10, ColorMagenta)
				continue // enemy consumed on hit
			}
		}

		aliveEnemies = append(aliveEnemies, e)
	}
	g.enemies = aliveEnemies
	if g.scene != ScenePlay {
		return
	}

	// 4b. Power-ups: bob, expire, collect
	alivePowers := g.powerups[:0]
	for _, p := range g.powerups {
		p.Life--
		p.Bob += 0.08
		if math.Hypot(g.player.X-p.X, g.player.Y-p.Y) < g.player.Radius+11 {
			g.collectPowerUp(p)
			continue
		}
		if p.Life > 0 {
			alivePowers = append(alivePowers, p)
		}
	}
	g.powerups = alivePowers

	// 5. Update Boss ZEUS.EXE
	if g.boss.Active {
		g.boss.Angle += 0.02
		// Smooth hover movement
		g.boss.X = ScreenWidth/2 + math.Cos(float64(g.timerFrames)*0.025)*160
		g.boss.Y = 110 + math.Sin(float64(g.timerFrames)*0.04)*30

		if g.boss.HitFlash > 0 {
			g.boss.HitFlash--
		}

		// Boss hit by player bolts
		for _, b := range g.bolts {
			if b.Life > 0 && math.Hypot(b.X-g.boss.X, b.Y-g.boss.Y) < g.boss.Radius+4 {
				b.Life = 0
				g.boss.HP--
				g.boss.HitFlash = 5
				g.score += 5
				g.burst(b.X, b.Y, 8, ColorWhite)
				if g.boss.HP <= 0 {
					// Win!
					g.scene = SceneWin
					g.sound.play(g.sound.winPCM)
					g.burst(g.boss.X, g.boss.Y, 80, ColorGold)
					return
				}
			}
		}

		// Enrage at half HP: summon minions
		if !g.boss.Enraged && g.boss.HP <= g.boss.MaxHP/2 {
			g.boss.Enraged = true
			g.boss.MinionTimer = 30
			g.shakeTimer = 14
			g.shakeMag = 5
			g.addText(g.boss.X, g.boss.Y+g.boss.Radius+16, "ZEUS ENRAGED!", ColorMagenta)
			g.sound.play(g.sound.thunderPCM)
		}
		if g.boss.Enraged {
			g.boss.MinionTimer--
			if g.boss.MinionTimer <= 0 {
				g.boss.MinionTimer = 100
				g.spawnEnemy(pickEnemyKind(BossSpawnSec))
			}
		}

		// Boss attacks: marks 6 random spots with warning circles for 0.8s (48 frames)
		g.boss.AttackTimer--
		if g.boss.AttackTimer <= 0 {
			g.boss.AttackTimer = 160 // Next attack cycle (~2.6s)
			// Pick 6 spots
			for i := 0; i < 6; i++ {
				// Mix of target near player and arena spread
				var sx, sy float64
				if i < 2 {
					sx = g.player.X + (rand.Float64()*80 - 40)
					sy = g.player.Y + (rand.Float64()*80 - 40)
				} else {
					sx = 40 + rand.Float64()*(ScreenWidth-80)
					sy = 50 + rand.Float64()*(ScreenHeight-90)
				}
				g.warnings = append(g.warnings, &WarningSpot{
					X:       sx,
					Y:       sy,
					Timer:   48, // 0.8s
					MaxTime: 48,
					Radius:  26,
				})
			}
		}
	}

	// 6. Update Warning Circles & Lightning Strikes
	activeWarnings := g.warnings[:0]
	for _, w := range g.warnings {
		w.Timer--
		if w.Timer <= 0 {
			// Strike!
			g.shakeTimer = 8
			g.shakeMag = 4.0
			g.sound.play(g.sound.thunderPCM)
			ls := &LightningStrike{
				X:    w.X,
				Y:    w.Y,
				Life: 10,
			}
			for j := 0; j < 6; j++ {
				ls.Jitter[j] = float32(rand.Float64()*24 - 12)
			}
			g.lightnings = append(g.lightnings, ls)
			g.burst(w.X, w.Y, 18, ColorGold)

			// Lightning also fries enemies caught inside (killed next frame)
			for _, e := range g.enemies {
				if math.Hypot(e.X-w.X, e.Y-w.Y) < w.Radius+e.Radius {
					e.HP = 0
				}
			}

			// Damage player if caught
			if !g.playerImmune() && math.Hypot(g.player.X-w.X, g.player.Y-w.Y) < w.Radius {
				g.playerTakeDamage()
			}
		} else {
			activeWarnings = append(activeWarnings, w)
		}
	}
	g.warnings = activeWarnings

	// Update lightning visual duration
	activeLightning := g.lightnings[:0]
	for _, l := range g.lightnings {
		l.Life--
		if l.Life > 0 {
			activeLightning = append(activeLightning, l)
		}
	}
	g.lightnings = activeLightning
}

func (g *Game) playerTakeDamage() {
	g.player.HP--
	g.player.InvulnTimer = 60 // 1s invulnerability
	g.shakeTimer = 16
	g.shakeMag = 6.5
	g.sound.play(g.sound.hitPCM)
	g.burst(g.player.X, g.player.Y, 20, ColorCyan)
	if g.player.HP <= 0 {
		g.scene = SceneGameOver
	}
}

func pickEnemyKind(elapsed float64) EnemyKind {
	r := rand.Float64()
	if elapsed > 8 && r < 0.15 {
		return EnemyMinotaur
	}
	if elapsed > 4 && r < 0.40 {
		return EnemyHarpy
	}
	return EnemyNode
}

func (g *Game) spawnEnemy(kind EnemyKind) {
	side := rand.Intn(4)
	var x, y float64
	switch side {
	case 0: // Top
		x = rand.Float64() * ScreenWidth
		y = -10
	case 1: // Bottom
		x = rand.Float64() * ScreenWidth
		y = ScreenHeight + 10
	case 2: // Left
		x = -10
		y = rand.Float64() * ScreenHeight
	case 3: // Right
		x = ScreenWidth + 10
		y = rand.Float64() * ScreenHeight
	}

	e := &Enemy{
		Kind:  kind,
		X:     x,
		Y:     y,
		Phase: rand.Float64() * math.Pi * 2,
		Side:  1,
	}
	if rand.Intn(2) == 0 {
		e.Side = -1
	}
	switch kind {
	case EnemyNode:
		e.Radius = 8.5
		e.Speed = 1.35 + rand.Float64()*0.4
		e.HP = 1
	case EnemyHarpy:
		e.Radius = 7
		e.Speed = 2.2 + rand.Float64()*0.3
		e.HP = 1
	case EnemyMinotaur:
		e.Radius = 13
		e.Speed = 0.85
		e.HP = 3
		e.ChargeCD = 60
	}
	e.MaxHP = e.HP
	g.enemies = append(g.enemies, e)
}

func (g *Game) killEnemy(e *Enemy) {
	dropChance := 0.10
	switch e.Kind {
	case EnemyNode:
		g.score += 10
		g.burst(e.X, e.Y, 18, ColorCyan)
	case EnemyHarpy:
		g.score += 15
		g.burst(e.X, e.Y, 14, ColorMagenta)
	case EnemyMinotaur:
		g.score += 30
		g.burst(e.X, e.Y, 30, ColorGold)
		dropChance = 0.35
	}
	g.sound.play(g.sound.enemyDeadPCM)
	if rand.Float64() < dropChance && len(g.powerups) < MaxPowerUps+1 &&
		e.X > 20 && e.X < ScreenWidth-20 && e.Y > 40 && e.Y < ScreenHeight-20 {
		g.spawnPowerUp(e.X, e.Y, randomPowerKind())
	}
}

// ---------- Power-ups ----------

func randomPowerKind() PowerUpKind {
	r := rand.Float64()
	switch {
	case r < 0.40:
		return PowerHeal
	case r < 0.70:
		return PowerShield
	default:
		return PowerRapid
	}
}

func (g *Game) spawnPowerUp(x, y float64, kind PowerUpKind) {
	g.powerups = append(g.powerups, &PowerUp{
		X:    x,
		Y:    y,
		Kind: kind,
		Life: PowerUpLife,
		Bob:  rand.Float64() * math.Pi * 2,
	})
}

func (g *Game) collectPowerUp(p *PowerUp) {
	switch p.Kind {
	case PowerHeal:
		if g.player.HP < g.player.MaxHP {
			g.player.HP++
			g.addText(p.X, p.Y, "+1 HP", ColorCyan)
		} else {
			g.score += 50
			g.addText(p.X, p.Y, "OVERHEAL +50", ColorMarble)
		}
		g.sound.play(g.sound.pickupPCM)
		g.burst(p.X, p.Y, 16, ColorCyan)
	case PowerShield:
		g.player.ShieldTimer = ShieldFrames
		g.addText(p.X, p.Y, "AEGIS SHIELD!", ColorGold)
		g.sound.play(g.sound.shieldPCM)
		g.burst(p.X, p.Y, 22, ColorGold)
	case PowerRapid:
		g.player.RapidTimer = RapidFrames
		g.addText(p.X, p.Y, "TRIPLE SHOT!", ColorCyan)
		g.sound.play(g.sound.pickupPCM)
		g.burst(p.X, p.Y, 16, ColorMarble)
	}
}

func (g *Game) addText(x, y float64, msg string, clr color.RGBA) {
	g.texts = append(g.texts, &FloatText{X: x, Y: y - 14, Msg: msg, Life: 60, MaxLife: 60, Color: clr})
}

func (g *Game) burst(x, y float64, count int, clr color.RGBA) {
	for i := 0; i < count; i++ {
		ang := rand.Float64() * math.Pi * 2
		spd := rand.Float64()*3.2 + 1.0
		g.particles = append(g.particles, &Particle{
			X:       x,
			Y:       y,
			VX:      math.Cos(ang) * spd,
			VY:      math.Sin(ang) * spd,
			Life:    18 + rand.Intn(12),
			MaxLife: 30,
			Size:    float32(rand.Float64()*2.5 + 1.5),
			Color:   clr,
		})
	}
}

// ---------- Drawing helpers ----------

type pt = [2]float32

// rot returns a function mapping local (lx, ly) coords rotated by angle a around (cx, cy).
func rot(a float64, cx, cy float32) func(lx, ly float64) pt {
	c, s := math.Cos(a), math.Sin(a)
	return func(lx, ly float64) pt {
		return pt{cx + float32(lx*c-ly*s), cy + float32(lx*s+ly*c)}
	}
}

func (g *Game) fillPoly(dst *ebiten.Image, pts []pt, clr color.Color) {
	if len(pts) < 3 {
		return
	}
	var p vector.Path
	p.MoveTo(pts[0][0], pts[0][1])
	for _, q := range pts[1:] {
		p.LineTo(q[0], q[1])
	}
	p.Close()
	g.vs, g.is = p.AppendVerticesAndIndicesForFilling(g.vs[:0], g.is[:0])
	r, gg, b, a := clr.RGBA()
	for i := range g.vs {
		g.vs[i].SrcX = 1
		g.vs[i].SrcY = 1
		g.vs[i].ColorR = float32(r) / 0xffff
		g.vs[i].ColorG = float32(gg) / 0xffff
		g.vs[i].ColorB = float32(b) / 0xffff
		g.vs[i].ColorA = float32(a) / 0xffff
	}
	op := &ebiten.DrawTrianglesOptions{
		AntiAlias:      true,
		ColorScaleMode: ebiten.ColorScaleModePremultipliedAlpha,
	}
	dst.DrawTriangles(g.vs, g.is, g.whitePixel, op)
}

func strokePoly(dst *ebiten.Image, pts []pt, width float32, clr color.Color) {
	for i := range pts {
		n := pts[(i+1)%len(pts)]
		vector.StrokeLine(dst, pts[i][0], pts[i][1], n[0], n[1], width, clr, true)
	}
}

func regularPoly(cx, cy float32, r float64, sides int, angle float64) []pt {
	pts := make([]pt, sides)
	for k := 0; k < sides; k++ {
		a := angle + float64(k)*2*math.Pi/float64(sides)
		pts[k] = pt{cx + float32(math.Cos(a)*r), cy + float32(math.Sin(a)*r)}
	}
	return pts
}

func fade(c color.RGBA, alpha float64) color.RGBA {
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	return color.RGBA{
		R: uint8(float64(c.R) * alpha),
		G: uint8(float64(c.G) * alpha),
		B: uint8(float64(c.B) * alpha),
		A: uint8(float64(c.A) * alpha),
	}
}

// ---------- Draw ----------

func (g *Game) Draw(screen *ebiten.Image) {
	target := g.canvas
	target.Fill(ColorBg)

	// Mosaic floor tiles (some cracked)
	for _, t := range g.tiles {
		vector.DrawFilledRect(target, t.X+1, t.Y+1, 30, 30, color.NRGBA{R: 0xED, G: 0xEB, B: 0xE6, A: t.Alpha}, false)
		if t.Crack {
			crack := color.NRGBA{R: 0x0B, G: 0x0F, B: 0x1A, A: 160}
			vector.StrokeLine(target, t.X+6, t.Y+4, t.X+15, t.Y+16, 1, crack, true)
			vector.StrokeLine(target, t.X+15, t.Y+16, t.X+12, t.Y+27, 1, crack, true)
			vector.StrokeLine(target, t.X+15, t.Y+16, t.X+26, t.Y+20, 1, crack, true)
		}
	}

	// Faint cyan grid on the floor
	gridClr := color.NRGBA{R: 0x3D, G: 0xF5, B: 0xFF, A: 28}
	for gx := 0.0; gx <= ScreenWidth; gx += 32 {
		vector.StrokeLine(target, float32(gx), 0, float32(gx), ScreenHeight, 1, gridClr, false)
	}
	for gy := 0.0; gy <= ScreenHeight; gy += 32 {
		vector.StrokeLine(target, 0, float32(gy), ScreenWidth, float32(gy), 1, gridClr, false)
	}

	// Arena border (Marble #EDEBE6)
	vector.StrokeRect(target, 8, 24, ScreenWidth-16, ScreenHeight-32, 2, ColorMarble, false)

	// White "column" rectangles along arena edges
	for _, cx := range []float32{60, 140, 220, 300, 380, 460, 540} {
		vector.DrawFilledRect(target, cx, 24, 14, 10, ColorMarble, false)
		vector.StrokeRect(target, cx, 24, 14, 10, 1, ColorWhite, false)
		vector.DrawFilledRect(target, cx, ScreenHeight-34, 14, 10, ColorMarble, false)
		vector.StrokeRect(target, cx, ScreenHeight-34, 14, 10, 1, ColorWhite, false)
	}
	for _, cy := range []float32{75, 145, 215, 285} {
		vector.DrawFilledRect(target, 8, cy, 10, 14, ColorMarble, false)
		vector.StrokeRect(target, 8, cy, 10, 14, 1, ColorWhite, false)
		vector.DrawFilledRect(target, ScreenWidth-18, cy, 10, 14, ColorMarble, false)
		vector.StrokeRect(target, ScreenWidth-18, cy, 10, 14, 1, ColorWhite, false)
	}

	// Terrain obstacles
	for _, o := range g.obstacles {
		g.drawObstacle(target, o)
	}

	// Warning circles for Zeus attacks (0.8s telegraph)
	for _, w := range g.warnings {
		prog := float32(w.Timer) / float32(w.MaxTime)
		warnClr := color.RGBA{R: 0xFF, G: 0xC9, B: 0x4A, A: uint8(140 + 115*(1-prog))}
		vector.StrokeCircle(target, float32(w.X), float32(w.Y), float32(w.Radius), 2, warnClr, true)
		// Shrinking inner lock circle
		vector.DrawFilledCircle(target, float32(w.X), float32(w.Y), float32(w.Radius)*(1.0-prog), color.RGBA{0xFF, 0x3D, 0x7F, 60}, true)
	}

	// Power-ups
	for _, p := range g.powerups {
		g.drawPowerUp(target, p)
	}

	// Enemies
	for _, e := range g.enemies {
		g.drawEnemy(target, e)
	}

	// Cyan Bolts
	for _, b := range g.bolts {
		bx := float32(b.X)
		by := float32(b.Y)
		vector.StrokeLine(target, bx, by, bx-float32(b.VX*1.2), by-float32(b.VY*1.2), 3, ColorCyan, true)
		vector.DrawFilledCircle(target, bx, by, 3, ColorMarble, true)
	}

	// Boss: ZEUS.EXE
	if g.boss.Active {
		g.drawBoss(target)
	}

	// Lightning strikes (jagged gold lines from sky)
	for _, l := range g.lightnings {
		lx := float32(l.X)
		ly := float32(l.Y)
		pts := []struct{ x, y float32 }{
			{lx, 0},
			{lx + l.Jitter[0], ly * 0.2},
			{lx + l.Jitter[1], ly * 0.4},
			{lx + l.Jitter[2], ly * 0.6},
			{lx + l.Jitter[3], ly * 0.8},
			{lx, ly},
		}
		for i := 0; i < len(pts)-1; i++ {
			vector.StrokeLine(target, pts[i].x, pts[i].y, pts[i+1].x, pts[i+1].y, 3.5, ColorGold, true)
			vector.StrokeLine(target, pts[i].x, pts[i].y, pts[i+1].x, pts[i+1].y, 1.5, ColorMarble, true)
		}
		vector.DrawFilledCircle(target, lx, ly, 22, color.RGBA{0xFF, 0xC9, 0x4A, 180}, true)
	}

	// Particles
	for _, p := range g.particles {
		alpha := float32(p.Life) / float32(p.MaxLife)
		c := p.Color
		c.A = uint8(float32(c.A) * alpha)
		vector.DrawFilledCircle(target, float32(p.X), float32(p.Y), p.Size*alpha, c, true)
	}

	// Player Drone
	if g.scene == ScenePlay && (g.player.InvulnTimer%6 < 3) {
		g.drawPlayer(target)
	}

	// Floating texts
	for _, t := range g.texts {
		a := float64(t.Life) / float64(t.MaxLife)
		w, _ := text.Measure(t.Msg, g.font, 0)
		g.drawText(target, t.Msg, t.X-w/2, t.Y, 1.0, fade(t.Color, a*1.5))
	}

	// Top HUD: HP, Timer, Score
	g.drawHUD(target)

	// Scene Overlays
	switch g.scene {
	case SceneTitle:
		g.drawTitleOverlay(target)
	case SceneGameOver:
		g.drawGameOverOverlay(target)
	case SceneWin:
		g.drawWinOverlay(target)
	}

	// Blit target canvas to screen with screen shake
	screen.Fill(ColorBg)
	g.drawOp.GeoM.Reset()
	if g.shakeTimer > 0 {
		ox := (rand.Float64() - 0.5) * g.shakeMag
		oy := (rand.Float64() - 0.5) * g.shakeMag
		g.drawOp.GeoM.Translate(ox, oy)
	}
	screen.DrawImage(g.canvas, g.drawOp)
}

func (g *Game) drawObstacle(dst *ebiten.Image, o *Obstacle) {
	x, y, r := float32(o.X), float32(o.Y), float32(o.R)
	// Drop shadow
	vector.DrawFilledCircle(dst, x+4, y+5, r, ColorShadow, true)

	switch o.Kind {
	case ObstacleColumn:
		// Top-down view of a Doric column capital
		vector.DrawFilledCircle(dst, x, y, r, ColorMarbleDark, true)
		vector.DrawFilledCircle(dst, x, y, r*0.86, ColorMarble, true)
		vector.StrokeCircle(dst, x, y, r, 1.5, ColorWhite, true)
		for k := 0; k < 10; k++ {
			a := o.Seed + float64(k)*math.Pi/5
			c, s := float32(math.Cos(a)), float32(math.Sin(a))
			vector.StrokeLine(dst, x+c*r*0.5, y+s*r*0.5, x+c*r*0.84, y+s*r*0.84, 1, ColorMarbleDark, true)
		}
		vector.StrokeCircle(dst, x, y, r*0.45, 1.5, ColorMarbleDark, true)
		vector.DrawFilledCircle(dst, x, y, 2.5, ColorGold, true)

	case ObstacleRubble:
		// Broken column with scattered fragments
		for k := 0; k < 4; k++ {
			a := o.Seed + float64(k)*1.7
			fx := x + float32(math.Cos(a))*r*1.05
			fy := y + float32(math.Sin(a))*r*1.05
			g.fillPoly(dst, regularPoly(fx, fy, 3.5+float64(k%2)*1.5, 4, a), ColorMarbleDark)
		}
		jag := make([]pt, 9)
		for k := range jag {
			a := o.Seed + float64(k)*2*math.Pi/9
			rr := float64(r) * (0.78 + 0.22*math.Abs(math.Sin(o.Seed*3+float64(k)*1.9)))
			jag[k] = pt{x + float32(math.Cos(a)*rr), y + float32(math.Sin(a)*rr)}
		}
		g.fillPoly(dst, jag, ColorMarble)
		strokePoly(dst, jag, 1.2, ColorMarbleDark)
		vector.StrokeLine(dst, x-r*0.4, y-r*0.2, x+r*0.1, y+r*0.1, 1, ColorMarbleDark, true)
		vector.StrokeLine(dst, x+r*0.1, y+r*0.1, x+r*0.3, y+r*0.5, 1, ColorMarbleDark, true)

	case ObstacleServer:
		// Server monolith with blinking LEDs
		h := r * 0.85
		vector.DrawFilledRect(dst, x-h, y-h, h*2, h*2, ColorSteel, true)
		vector.StrokeRect(dst, x-h, y-h, h*2, h*2, 1.5, ColorCyan, true)
		for row := 0; row < 4; row++ {
			ly := y - h + 4 + float32(row)*(h*2-8)/3
			vector.StrokeLine(dst, x-h+3, ly, x+h-9, ly, 1, color.RGBA{0x3D, 0xF5, 0xFF, 70}, true)
			on := (g.tick/12+row*3+int(o.Seed*10))%5 != 0
			led := ColorCyan
			if (row+int(o.Seed*7))%3 == 0 {
				led = ColorMagenta
			}
			if !on {
				led = ColorSteel
			}
			vector.DrawFilledRect(dst, x+h-7, ly-1.5, 3, 3, led, false)
		}
	}
}

func (g *Game) drawPowerUp(dst *ebiten.Image, p *PowerUp) {
	if p.Life < 120 && (g.tick/4)%2 == 0 {
		return // blink before vanishing
	}
	x := float32(p.X)
	y := float32(p.Y + math.Sin(p.Bob)*2.5)
	pulse := float32(math.Sin(float64(g.tick)*0.12)) * 2

	switch p.Kind {
	case PowerHeal: // Ambrosia
		vector.DrawFilledCircle(dst, x, y, 14+pulse, color.RGBA{0x3D, 0xF5, 0xFF, 40}, true)
		vector.DrawFilledCircle(dst, x, y, 9, ColorMarble, true)
		vector.StrokeCircle(dst, x, y, 9, 1.5, ColorCyan, true)
		vector.DrawFilledRect(dst, x-1.75, y-5.5, 3.5, 11, ColorCyan, true)
		vector.DrawFilledRect(dst, x-5.5, y-1.75, 11, 3.5, ColorCyan, true)
	case PowerShield: // Aegis
		vector.DrawFilledCircle(dst, x, y, 15+pulse, color.RGBA{0xFF, 0xC9, 0x4A, 45}, true)
		hex := regularPoly(x, y, 10, 6, math.Pi/6)
		g.fillPoly(dst, hex, ColorGold)
		g.fillPoly(dst, regularPoly(x, y, 6, 6, math.Pi/6), ColorMarble)
		vector.DrawFilledCircle(dst, x, y, 2.5, ColorBg, true)
		strokePoly(dst, hex, 1.2, ColorWhite)
	case PowerRapid: // Owl Sight
		vector.DrawFilledCircle(dst, x, y, 14+pulse, color.RGBA{0x3D, 0xF5, 0xFF, 40}, true)
		vector.DrawFilledCircle(dst, x, y, 9, ColorBg, true)
		vector.StrokeCircle(dst, x, y, 9, 2, ColorCyan, true)
		for _, a := range []float64{-0.55, 0, 0.55} {
			ang := -math.Pi/2 + a
			ex := x + float32(math.Cos(ang)*6)
			ey := y + float32(math.Sin(ang)*6)
			vector.StrokeLine(dst, x, y+3, ex, ey, 1.5, ColorCyan, true)
			vector.DrawFilledCircle(dst, ex, ey, 1.5, ColorMarble, true)
		}
	}
}

func (g *Game) drawEnemy(dst *ebiten.Image, e *Enemy) {
	ex, ey := float32(e.X), float32(e.Y)
	r := float64(e.Radius)
	flash := e.HitFlash > 0

	switch e.Kind {
	case EnemyNode:
		// Spinning purge node with spikes
		body := ColorMagenta
		if flash {
			body = ColorWhite
		}
		for k := 0; k < 6; k++ {
			a := e.Phase*0.3 + float64(k)*math.Pi/3
			c, s := float32(math.Cos(a)), float32(math.Sin(a))
			vector.StrokeLine(dst, ex+c*float32(r*0.8), ey+s*float32(r*0.8), ex+c*float32(r+4), ey+s*float32(r+4), 2, body, true)
		}
		vector.DrawFilledCircle(dst, ex, ey, float32(r), body, true)
		vector.StrokeCircle(dst, ex, ey, float32(r), 1.5, ColorMarble, true)
		vector.DrawFilledCircle(dst, ex, ey, float32(r*0.45), ColorBg, true)
		core := float32(1.5 + math.Sin(e.Phase)*0.8)
		vector.DrawFilledCircle(dst, ex, ey, core, ColorMagenta, true)

	case EnemyHarpy:
		// Fast winged bot, points where it flies
		P := rot(e.Heading, ex, ey)
		flap := math.Sin(e.Phase*2) * 4
		wing := ColorMagentaDark
		body := ColorMagenta
		if flash {
			wing, body = ColorWhite, ColorWhite
		}
		g.fillPoly(dst, []pt{P(2, -2), P(-6, -11-flap), P(-3, -2)}, wing)
		g.fillPoly(dst, []pt{P(2, 2), P(-6, 11+flap), P(-3, 2)}, wing)
		hull := []pt{P(10, 0), P(-6, -5), P(-3, 0), P(-6, 5)}
		g.fillPoly(dst, hull, body)
		strokePoly(dst, hull, 1, ColorMarble)
		eye := P(3, 0)
		vector.DrawFilledCircle(dst, eye[0], eye[1], 1.8, ColorGold, true)

	case EnemyMinotaur:
		// Heavy horned brute
		P := rot(e.Heading, ex, ey)
		body := ColorMagentaDark
		edge := ColorMagenta
		if flash {
			body, edge = ColorWhite, ColorWhite
		}
		if e.ChargeTimer > 0 {
			// Charge trail
			back := P(-r-10, 0)
			vector.StrokeLine(dst, ex, ey, back[0], back[1], 6, color.RGBA{0xFF, 0x3D, 0x7F, 70}, true)
		}
		oct := make([]pt, 8)
		for k := range oct {
			a := float64(k)*math.Pi/4 + math.Pi/8
			oct[k] = P(math.Cos(a)*r, math.Sin(a)*r)
		}
		g.fillPoly(dst, oct, body)
		strokePoly(dst, oct, 2, edge)
		// Horns
		for _, s := range []float64{-1, 1} {
			a1, a2, a3 := P(5, 8*s), P(12, 15*s), P(17, 11*s)
			vector.StrokeLine(dst, a1[0], a1[1], a2[0], a2[1], 2.5, ColorGold, true)
			vector.StrokeLine(dst, a2[0], a2[1], a3[0], a3[1], 2, ColorGold, true)
		}
		// Eyes glow white when about to / while charging
		eyeClr := ColorGold
		if e.ChargeTimer > 0 || (e.ChargeCD == 0) {
			eyeClr = ColorWhite
		}
		for _, s := range []float64{-1, 1} {
			p := P(6, 4*s)
			vector.DrawFilledCircle(dst, p[0], p[1], 2, eyeClr, true)
		}
		// HP pips
		for i := 0; i < e.MaxHP; i++ {
			px := ex - float32(e.MaxHP*6)/2 + float32(i*6)
			py := ey - float32(r) - 9
			if i < e.HP {
				vector.DrawFilledRect(dst, px, py, 4, 3, ColorMagenta, false)
			} else {
				vector.StrokeRect(dst, px, py, 4, 3, 1, ColorMarbleDark, false)
			}
		}
	}
}

func (g *Game) drawPlayer(dst *ebiten.Image) {
	px := float32(g.player.X)
	py := float32(g.player.Y)
	P := rot(g.player.Angle, px, py)

	// Shadow
	vector.DrawFilledCircle(dst, px+3, py+5, 11, ColorShadow, true)

	// Aegis shield ring
	if g.player.ShieldTimer > 0 && (g.player.ShieldTimer > 90 || (g.tick/5)%2 == 0) {
		vector.DrawFilledCircle(dst, px, py, 21, color.RGBA{0xFF, 0xC9, 0x4A, 35}, true)
		strokePoly(dst, regularPoly(px, py, 21, 6, float64(g.tick)*0.05), 2, ColorGold)
	}
	// Owl Sight orbiters
	if g.player.RapidTimer > 0 {
		for k := 0; k < 3; k++ {
			a := float64(g.tick)*0.15 + float64(k)*2*math.Pi/3
			vector.DrawFilledCircle(dst, px+float32(math.Cos(a)*16), py+float32(math.Sin(a)*16), 2, ColorCyan, true)
		}
	}

	// Thruster flame
	fl := 6 + 3*math.Sin(float64(g.tick)*0.7)
	g.fillPoly(dst, []pt{P(-7, -3.5), P(-7-fl, 0), P(-7, 3.5)}, ColorCyan)

	// Swept wings
	lw := []pt{P(3, -4), P(-6, -15), P(-10, -13), P(-5, -3)}
	rw := []pt{P(3, 4), P(-6, 15), P(-10, 13), P(-5, 3)}
	g.fillPoly(dst, lw, ColorMarbleDark)
	g.fillPoly(dst, rw, ColorMarbleDark)
	strokePoly(dst, lw, 1, ColorCyan)
	strokePoly(dst, rw, 1, ColorCyan)
	for _, s := range []float64{-1, 1} {
		tip := P(-8, 14*s)
		vector.DrawFilledCircle(dst, tip[0], tip[1], 1.8, ColorGold, true)
	}

	// Hull
	hull := []pt{P(16, 0), P(5, -7), P(-8, -6), P(-8, 6), P(5, 7)}
	g.fillPoly(dst, hull, ColorMarble)
	strokePoly(dst, hull, 1.5, ColorCyan)

	// Owl eyes
	for _, s := range []float64{-1, 1} {
		e := P(4, 3.3*s)
		vector.DrawFilledCircle(dst, e[0], e[1], 3, ColorCyan, true)
		pp := P(5, 3.3*s)
		vector.DrawFilledCircle(dst, pp[0], pp[1], 1.3, ColorBg, true)
	}
	beak := []pt{P(9, 0), P(6.5, -1.5), P(6.5, 1.5)}
	g.fillPoly(dst, beak, ColorGold)
}

func (g *Game) drawBoss(target *ebiten.Image) {
	bx := float32(g.boss.X)
	by := float32(g.boss.Y)
	r := float32(g.boss.Radius)
	flashing := g.boss.HitFlash > 0

	// Pulsing aura (magenta when enraged)
	auraClr := color.RGBA{0xFF, 0xC9, 0x4A, 90}
	if g.boss.Enraged {
		auraClr = color.RGBA{0xFF, 0x3D, 0x7F, 110}
	}
	aura := r + 9 + float32(math.Sin(float64(g.tick)*0.1))*3
	vector.StrokeCircle(target, bx, by, aura, 2, auraClr, true)

	bossColor := ColorGold
	if flashing {
		bossColor = ColorWhite
		vector.DrawFilledCircle(target, bx, by, r, ColorWhite, true)
	}

	// Crown of lightning spikes
	for k := -2; k <= 2; k++ {
		cx := bx + float32(k)*9
		h := float32(14 - 3*absInt(k))
		g.fillPoly(target, []pt{{cx - 4, by - r + 4}, {cx, by - r - h}, {cx + 4, by - r + 4}}, bossColor)
	}

	// Hexagon body
	var hexPts [6][2]float32
	for k := 0; k < 6; k++ {
		ang := g.boss.Angle + float64(k)*math.Pi/3.0
		hexPts[k][0] = bx + float32(math.Cos(ang))*r
		hexPts[k][1] = by + float32(math.Sin(ang))*r
	}
	for k := 0; k < 6; k++ {
		next := (k + 1) % 6
		vector.StrokeLine(target, hexPts[k][0], hexPts[k][1], hexPts[next][0], hexPts[next][1], 3.5, bossColor, true)
		vector.StrokeLine(target, bx, by, hexPts[k][0], hexPts[k][1], 1.5, bossColor, true)
	}

	// Marble beard (zig-zag strands)
	for s := -1; s <= 1; s++ {
		sx := bx + float32(s)*9
		y0 := by + r*0.45
		prev := pt{sx, y0}
		for j := 1; j <= 4; j++ {
			zig := float32(3)
			if j%2 == 0 {
				zig = -3
			}
			cur := pt{sx + zig - float32(s)*float32(j), y0 + float32(j)*5}
			vector.StrokeLine(target, prev[0], prev[1], cur[0], cur[1], 2, ColorMarble, true)
			prev = cur
		}
	}

	// Core: magenta while charging a volley
	coreColor := ColorCyan
	if g.boss.AttackTimer < 30 {
		coreColor = ColorMagenta
	}
	if flashing {
		coreColor = ColorWhite
	}
	vector.DrawFilledCircle(target, bx, by, r*0.42, coreColor, true)
	vector.StrokeCircle(target, bx, by, r*0.55, 2, ColorMarble, true)

	// Eyes that track the drone
	look := math.Atan2(g.player.Y-g.boss.Y, g.player.X-g.boss.X)
	lx, ly := float32(math.Cos(look)*2.5), float32(math.Sin(look)*2.5)
	for _, s := range []float32{-1, 1} {
		vector.DrawFilledRect(target, bx+s*6-3+lx, by-3+ly, 6, 3, ColorBg, true)
		vector.DrawFilledRect(target, bx+s*6-1.5+lx, by-2.5+ly, 3, 2, ColorWhite, true)
	}

	// Boss HP Bar at Top
	barW := float32(280)
	barH := float32(12)
	barX := float32(ScreenWidth/2) - barW/2
	barY := float32(32)
	vector.StrokeRect(target, barX, barY, barW, barH, 2, ColorGold, false)
	fillRatio := float32(g.boss.HP) / float32(g.boss.MaxHP)
	if fillRatio > 0 {
		fill := ColorGold
		if g.boss.Enraged {
			fill = ColorMagenta
		}
		vector.DrawFilledRect(target, barX+2, barY+2, (barW-4)*fillRatio, barH-4, fill, false)
	}
	g.drawTextCentered(target, "BOSS: ZEUS.EXE [SERVER-CITADEL ROOT]", 22, 1.0, ColorGold)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func (g *Game) drawHUD(screen *ebiten.Image) {
	// HP Pips
	g.drawText(screen, "HP:", 18, 12, 1.0, ColorMarble)
	for i := 0; i < g.player.MaxHP; i++ {
		pipX := float32(45 + i*14)
		if i < g.player.HP {
			vector.DrawFilledCircle(screen, pipX, 15, 5, ColorCyan, true)
		} else {
			vector.StrokeCircle(screen, pipX, 15, 5, 1.5, color.RGBA{0x60, 0x70, 0x80, 0xFF}, true)
		}
	}

	// Timer
	totalSec := g.timerFrames / 60
	timeStr := fmt.Sprintf("TIMER: %02d:%02d", totalSec/60, totalSec%60)
	if float64(totalSec) < BossSpawnSec && !g.boss.Active {
		timeStr += fmt.Sprintf(" (ZEUS IN %ds)", int(BossSpawnSec)-totalSec)
	}
	g.drawText(screen, timeStr, 220, 12, 1.0, ColorGold)

	// Score
	scoreStr := fmt.Sprintf("SCORE: %04d", g.score)
	g.drawText(screen, scoreStr, ScreenWidth-120, 12, 1.0, ColorMarble)

	// Active power-ups (bottom-left)
	if g.scene != ScenePlay {
		return
	}
	x := 20.0
	y := float64(ScreenHeight - 22)
	drawBuff := func(label string, frames, maxFrames int, clr color.RGBA) {
		g.drawText(screen, label, x, y, 1.0, clr)
		bx := float32(x) + float32(len(label)*7) + 6
		vector.StrokeRect(screen, bx, float32(y)+4, 52, 6, 1, clr, false)
		vector.DrawFilledRect(screen, bx+1, float32(y)+5, 50*float32(frames)/float32(maxFrames), 4, clr, false)
		x += float64(len(label)*7) + 72
	}
	if g.player.ShieldTimer > 0 {
		drawBuff("AEGIS", g.player.ShieldTimer, ShieldFrames, ColorGold)
	}
	if g.player.RapidTimer > 0 {
		drawBuff("OWL SIGHT", g.player.RapidTimer, RapidFrames, ColorCyan)
	}
}

func (g *Game) drawTitleOverlay(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 60, 40, ScreenWidth-120, ScreenHeight-80, color.RGBA{0x0B, 0x0F, 0x1A, 0xEE}, false)
	vector.StrokeRect(screen, 60, 40, ScreenWidth-120, ScreenHeight-80, 2, ColorCyan, false)

	g.drawTextCentered(screen, "OLYMPUS 2.0", 72, 2.4, ColorCyan)
	g.drawTextCentered(screen, "DEPRECATED GODS RUN THE SERVER-CITADEL ABOVE ATHENS", 112, 1.0, ColorMarble)
	g.drawTextCentered(screen, "YOU ARE A ROGUE DRONE CLIMBING TO STEAL ROOT ACCESS", 128, 1.0, ColorGold)

	g.drawTextCentered(screen, "CONTROLS: [W][A][S][D] Move | [MOUSE] Aim | [LEFT CLICK] Shoot", 160, 1.0, ColorCyan)
	g.drawTextCentered(screen, fmt.Sprintf("SURVIVE %d SECONDS OF NODES, HARPIES & MINOTAURS", int(BossSpawnSec)), 186, 1.1, ColorMagenta)
	g.drawTextCentered(screen, "THEN OVERLOAD & PURGE 'ZEUS.EXE' TO WIN", 204, 1.1, ColorGold)
	g.drawTextCentered(screen, "USE COLUMNS & SERVERS AS COVER - THEY BLOCK BOLTS BOTH WAYS", 228, 1.0, ColorMarble)
	g.drawTextCentered(screen, "PICKUPS: AMBROSIA +1 HP | AEGIS 6s IMMUNITY | OWL SIGHT TRIPLE SHOT", 246, 1.0, ColorCyan)

	// Pulsing prompt
	pulseAlpha := uint8(180 + 75*math.Sin(float64(g.tick)*0.1))
	g.drawTextCentered(screen, ">>> CLICK TO START <<<", 282, 1.5, color.RGBA{0x3D, 0xF5, 0xFF, pulseAlpha})
}

func (g *Game) drawGameOverOverlay(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 90, 70, ScreenWidth-180, ScreenHeight-140, color.RGBA{0x0B, 0x0F, 0x1A, 0xF2}, false)
	vector.StrokeRect(screen, 90, 70, ScreenWidth-180, ScreenHeight-140, 2, ColorMagenta, false)

	g.drawTextCentered(screen, "ROOT ACCESS DENIED", 115, 2.2, ColorMagenta)
	g.drawTextCentered(screen, "YOUR DRONE CORE WAS PURGED BY CITADEL DEFENSES", 155, 1.1, ColorMarble)
	g.drawTextCentered(screen, fmt.Sprintf("FINAL SCORE: %d", g.score), 185, 1.3, ColorGold)

	g.drawTextCentered(screen, ">>> CLICK TO RESTART <<<", 230, 1.4, ColorCyan)
}

func (g *Game) drawWinOverlay(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 70, 60, ScreenWidth-140, ScreenHeight-120, color.RGBA{0x0B, 0x0F, 0x1A, 0xF4}, false)
	vector.StrokeRect(screen, 70, 60, ScreenWidth-140, ScreenHeight-120, 2.5, ColorGold, false)

	g.drawTextCentered(screen, "ROOT ACCESS GRANTED!", 105, 2.2, ColorGold)
	g.drawTextCentered(screen, "ZEUS.EXE DEPRECATED & PURGED FROM SERVER-CITADEL", 145, 1.1, ColorCyan)
	g.drawTextCentered(screen, "ATHENS SKY IS LIBERATED FROM THE LEGACY OLYMPIANS", 168, 1.0, ColorMarble)
	g.drawTextCentered(screen, fmt.Sprintf("VICTORY SCORE: %d", g.score), 200, 1.4, ColorGold)

	g.drawTextCentered(screen, ">>> CLICK TO PLAY AGAIN <<<", 240, 1.4, ColorCyan)
}

func (g *Game) drawText(screen *ebiten.Image, msg string, x, y float64, scale float64, clr color.RGBA) {
	op := &text.DrawOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(screen, msg, g.font, op)
}

func (g *Game) drawTextCentered(screen *ebiten.Image, msg string, y float64, scale float64, clr color.RGBA) {
	w, _ := text.Measure(msg, g.font, 0)
	totalW := w * scale
	x := (float64(ScreenWidth) - totalW) / 2.0
	g.drawText(screen, msg, x, y, scale, clr)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func main() {
	ebiten.SetWindowSize(960, 540)
	ebiten.SetWindowTitle("OLYMPUS 2.0 — Rogue Drone Server Citadel Infiltration")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(60)

	g := NewGame()
	if err := ebiten.RunGame(g); err != nil {
		log.Fatalf("Olympus 2.0 crashed: %v", err)
	}
}
