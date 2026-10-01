# NetRasad — Asset Generation Prompts for Gemini

> This file contains prompts you can copy-paste into Gemini (or any image generation AI)
> to generate the visual assets needed for the NetRasad desktop application.
>
> After generating, save the files into the `assets/` directory and `frontend/src/assets/`.

---

## 1. App Logo / Icon (512×512 PNG, transparent background)

```
Create a modern, minimalist app icon for "NetRasad", a network monitoring application.

Design requirements:
- Square 512×512 pixels, transparent background, rounded corners feel
- A stylized "R" letterform combined with a network/signal motif
- Color palette: deep navy (#1B1F26) background, electric cyan (#00D4FF) and 
  emerald green (#00FF88) for the signal/activity elements
- The icon should convey: network activity, monitoring, data flow
- Style: flat design, clean geometric shapes, subtle gradient on the accent elements
- Think: a radar/scope aesthetic merged with a bandwidth graph
- No text other than the implied "R" shape
- Should look good at 32×32 (taskbar) and 512×512 (app icon)

Output: PNG with transparent background, 512×512
```

---

## 2. System Tray Icon (32×32 ICO/PNG)

```
Create a 32×32 pixel system tray icon for a network monitoring app called "NetRasad".

Design:
- Simple, recognizable at 16×16 and 32×32 sizes
- A small radar/signal arc with a dot in the center
- Monochrome white with slight transparency for dark taskbar
- OR: cyan (#00D4FF) signal arc on transparent background
- Must be clearly visible on both light and dark taskbars
- No text, no fine details — bold and simple

Output: PNG 32×32 with transparent background (will be converted to ICO)
```

---

## 3. Dashboard Background Pattern (subtle, 1920×1080)

```
Create a subtle, dark background pattern for a network monitoring dashboard.

Design:
- 1920×1080 pixels
- Base color: #1B1F26 (deep navy-charcoal)
- Faint geometric grid pattern (like a network graph) at 5-8% opacity
- Subtle hexagonal or circuit-board motif in the corners
- Should be barely visible — not distracting from data/charts
- Dark mode aesthetic, professional, technical

Output: PNG 1920×1080
```

---

## 4. Empty State Illustration — No Traffic (400×300 SVG/PNG)

```
Create a simple, flat-style illustration for an "no traffic data" empty state.

Design:
- 400×300 pixels
- A stylized router/network device with "zzz" or a flatline graph
- Colors: muted gray-blue tones (#3A4451, #5A6471) with a hint of cyan
- Minimalist, friendly, not alarming
- Flat design, no gradients, thin line art style
- Transparent or dark background (#1B1F26)

Output: SVG preferred, or PNG with transparent background
```

---

## 5. Empty State Illustration — No Interfaces (400×300 SVG/PNG)

```
Create a flat-style illustration for "no network interfaces found" empty state.

Design:
- 400×300 pixels
- A disconnected network cable or a sad router face
- Colors: muted gray-blue (#3A4451, #5A6471) with a small red/orange accent
- Minimalist, flat design, thin lines
- Dark background (#1B1F26) or transparent

Output: SVG preferred, or PNG with transparent background
```

---

## 6. Speed Test Animation Concept (for CSS/Canvas reference)

```
Create a reference image for a speed test gauge animation.

Design:
- A semicircular gauge (180 degrees) from left to right
- Background arc: dark gray (#2A3142)
- Foreground arc gradient: green (#00FF88) → yellow (#FFD700) → red (#FF4444)
- Needle pointing at ~70% position
- Center text area for "Download Speed" with a large number
- Clean, modern, dark theme
- 600×400 pixels

This is a reference for implementing the gauge in CSS/Canvas.

Output: PNG 600×400
```

---

## 7. App Splash Screen (1024×700)

```
Create a splash/loading screen for "NetRasad" network monitoring app.

Design:
- 1024×700 pixels
- Dark background (#1B1F26) with subtle network grid pattern
- Centered: the NetRasad logo (stylized R + signal motif) in cyan/green
- Below logo: "NetRasad" text in clean sans-serif, white
- Below text: "Network Monitoring Suite" in smaller, muted gray text
- A thin animated-looking loading bar at the bottom (cyan)
- Professional, modern, clean

Output: PNG 1024×700
```

---

## File Naming Convention After Generation

Save generated files as:
```
assets/
├── icon.png              (512×512 app icon)
├── tray-icon.png         (32×32 tray icon)
├── splash.png            (1024×700 splash)
├── empty-no-traffic.svg  (empty state)
├── empty-no-interfaces.svg (empty state)
└── speedtest-gauge-ref.png (reference)

frontend/src/assets/
├── logo.svg              (vector logo for UI header)
├── icon-small.png        (32×32 for UI)
└── pattern.png           (background pattern)
```

For Wails build icons, place in:
```
build/
├── appicon.png           (512×512, used by Wails for Windows .ico generation)
└── windows/
    └── icon.ico          (generated from appicon.png)
```