/** @type {import('tailwindcss').Config} */
export default {
  darkMode: "class",
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      fontFamily: {
        display: ["Bebas Neue", "system-ui", "sans-serif"],
        body: ["Manrope", "system-ui", "sans-serif"],
      },
      colors: {
        "brand-ink": "#0a0c10",
        "brand-sand": "#f4efe8",
        "brand-amber": "#f8b400",
        "brand-ember": "#ea5a4f",
        "brand-reef": "#0aa6b6",
        "brand-night": "#0b1d2b",
        "dark-ink": "#f4efe8",
        "dark-sand": "#0d1621",
        "dark-night": "#101a27",
      },
      boxShadow: {
        glow: "0 12px 40px rgba(10, 166, 182, 0.35)",
      },
    },
  },
  plugins: [],
}

