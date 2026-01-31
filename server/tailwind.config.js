/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./view/**/*.templ",
    "./view/*.templ"
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Kanit', 'sans-serif'],
      },
    },
  },
  plugins: [],
  darkMode: 'class',
}
