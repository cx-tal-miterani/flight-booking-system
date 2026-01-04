/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        midnight: '#0a0a0f',
        charcoal: '#141419',
        'custom-slate': '#1e1e28',
        border: 'hsl(var(--border))',
        foreground: 'hsl(var(--foreground))',
        background: 'hsl(var(--background))',
        accent: {
          cyan: '#00d4ff',
          emerald: '#10b981',
          amber: '#f59e0b',
          rose: '#f43f5e',
        },
      },
    },
  },
  plugins: [],
}

