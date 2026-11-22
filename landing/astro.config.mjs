import { defineConfig } from 'astro/config';

const base = process.env.ASTRO_BASE || '/';

export default defineConfig({
  site: 'https://aedmadrid.github.io',
  base,
  output: 'static',
});
