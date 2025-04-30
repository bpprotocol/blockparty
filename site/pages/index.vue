<template>
  <section class="hero is-fullheight-with-navbar is-light">
    <div class="hero-body">
      <div class="container has-text-centered">
        <h1 class="title is-1 mb-4">
          BlockParty Protocol
        </h1>
        <h2 class="subtitle is-4 mb-6">
          A Post-Censorship Social Protocol for the Network-Agnostic Era
        </h2>

        <div class="buttons is-centered mb-5">
          <NuxtLink to="/whitepaper" class="button is-primary is-medium">
            Read the Whitepaper
          </NuxtLink>
          <NuxtLink to="/manifesto" class="button is-light is-medium">
            Read the Manifesto
          </NuxtLink>
        </div>

        <div class="content has-text-centered mt-6">
          <p class="is-size-5">
            Build worlds. Burn identities. Speak freely.
          </p>
          <p class="is-size-6 mt-2 has-text-grey">
            Make protocols, not platforms.
          </p>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
useHead({
  title: 'BlockParty Protocol',
  meta: [
    { name: 'description', content: 'A Post-Censorship Social Protocol for the Network-Agnostic Era' },
    { property: 'og:title', content: 'BlockParty Protocol' },
    { property: 'og:description', content: 'A Post-Censorship Social Protocol for the Network-Agnostic Era' },
    { property: 'og:image', content: '/images/hero-image.jpg' },
    { property: 'og:url', content: 'https://bpprotocol.org' }
  ]
})
</script>

<style scoped>
.hero.is-fullheight-with-navbar {
  min-height: calc(100vh - 3.25rem); /* Adjust for navbar height */
}
</style>
