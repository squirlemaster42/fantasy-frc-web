/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./view/**/*.templ",
    "./view/*.templ"
  ],
  safelist: [
    'bg-slate-500',
    'bg-slate-600',
    'bg-zinc-500',
    'bg-stone-500',
    'bg-sky-600',
    'bg-blue-600',
    'bg-indigo-500',
    'bg-violet-500',
    'bg-teal-600',
    'bg-emerald-600',
    'bg-amber-700',
    'bg-rose-500',
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Kanit', 'sans-serif'],
      },
      colors: {
        'base-100': '#16181a',
        'base-200': '#1f2225',
        'base-300': '#2d3136',
        'base-content': '#e9e7e8',
        primary: '#7d94b2',
        'primary-content': '#16181a',
        secondary: '#8b919d',
        'secondary-content': '#16181a',
        accent: '#ac978c',
        'accent-content': '#16181a',
        neutral: '#5f5652',
        'neutral-content': '#e9e7e8',
        info: '#7d94b2',
        'info-content': '#16181a',
        success: '#85b885',
        'success-content': '#16181a',
        warning: '#c49a6a',
        'warning-content': '#16181a',
        error: '#c47a7a',
        'error-content': '#16181a',
      },
    },
  },
  plugins: [],
}
