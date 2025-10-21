/** @type {import('tailwindcss').Config} */
module.exports = {
    // CRITICAL: This 'content' array tells Tailwind where to find your utility classes.
    // It must include all file types where you write Tailwind classes.
    content: [
        "./index.html",
        "./src/**/*.{js,ts,jsx,tsx,svelte}",
    ],
    theme: {
        extend: {},
    },
    // We keep the DaisyUI plugin here, which is why those classes were working.
    plugins: [require("daisyui")],

    // Optional: DaisyUI configuration
    daisyui: {
        themes: ["light", "dark"],
    },
    mode: "jit",
}
