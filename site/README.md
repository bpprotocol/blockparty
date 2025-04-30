# BlockParty Website

This directory contains the source for [**bpprotocol.org**](https://bpprotocol.org), the canonical website for the BlockParty Protocol. It includes:

- Rendered versions of the whitepaper, specs, roadmap, and manifesto
- The Foundations essay anthology
- Community and project documentation
- Site content built with [Nuxt](https://nuxt.com)

---

## ✦ Setup

Install dependencies:

```bash
# npm
npm install

# pnpm
pnpm install

# yarn
yarn install

# bun
bun install
```

---

## ✦ Development

Start the local dev server on `http://localhost:3000`:

```bash
# npm
npm run dev

# pnpm
pnpm dev

# yarn
yarn dev

# bun
bun run dev
```

The site will automatically pull in markdown files from:
- `../protocol/`
- `../docs/`
- `../foundations/`

These are bundled into the rendered site during build.

---

## ✦ Production

To build for production:

```bash
# npm
npm run generate

# pnpm
pnpm generate

# yarn
yarn generate

# bun
bun run generate
```

To preview the production build locally:

```bash
# npm
npm run preview

# pnpm
pnpm preview

# yarn
yarn preview

# bun
bun run preview
```

---

## ✦ Deployment

This site is intended to be statically deployed. See [Nuxt deployment guide](https://nuxt.com/docs/getting-started/deployment) for options.

You may deploy to:
- GitHub Pages
- Vercel
- Cloudflare Pages
- IPFS
- Anywhere static hosting is supported

---

## ✦ Notes

- The site assumes relative paths to upstream protocol docs and essays.
- During the build step, it copies and transforms relevant `.md` files into Nuxt content format.

Pull requests welcome for copy edits, styling improvements, or new essays.