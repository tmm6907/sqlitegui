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
    safeList: [
        'alert-success',
        'alert-error',
        'alert-warning',
        'alert-info',
        // You can also use a pattern for all of them:
        {
            pattern: /alert-(success|error|warning|info)/,
            variants: ['bg', 'text', 'border'], // If you have variations
        },
    ],
    // Optional: DaisyUI configuration
    daisyui: {
        themes: ["light", "dark"],
    },
    mode: "jit",
}
