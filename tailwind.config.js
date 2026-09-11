/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./web/templates/**/*.html"],
  theme: {
    extend: {
      colors: {
        ink: {
          DEFAULT: "#14100c",
          soft: "#1e1812",
        },
        parchment: "#f3e6c9",
        gold: "#d4b45a",
      },
      fontFamily: {
        display: ['Palatino Linotype', "Palatino", "Georgia", "serif"],
        sans: ["Georgia", "Palatino Linotype", "Times New Roman", "serif"],
      },
      backgroundImage: {
        vignette: "radial-gradient(ellipse at center, transparent 40%, rgba(0,0,0,0.45) 100%)",
      },
    },
  },
  plugins: [],
};
