import { defineConfig } from 'astro/config';

export default defineConfig({
  site: 'https://openmimo.aedm.org.es',
  base: '/',
  output: 'static',
  // Astro 7 defaults to JSX whitespace rules, which can glue inline elements
  // together (e.g. <a>…</a> seguido de texto). Keep HTML-aware compression.
  compressHTML: true,
});
