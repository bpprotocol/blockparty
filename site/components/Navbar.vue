<template>
  <nav class="navbar is-dark" role="navigation" aria-label="main navigation">
    <div class="navbar-brand">
      <NuxtLink to="/" class="navbar-item">
        <span class="logo-text">BlockParty</span>
      </NuxtLink>

      <a role="button" class="navbar-burger" aria-label="menu" aria-expanded="false" @click="toggleNavbar">
        <span aria-hidden="true"></span>
        <span aria-hidden="true"></span>
        <span aria-hidden="true"></span>
      </a>
    </div>

    <div :class="['navbar-menu', { 'is-active': isNavbarActive }]">
      <div class="navbar-start">
        <NuxtLink to="/whitepaper" class="navbar-item">Whitepaper</NuxtLink>
        <NuxtLink to="/manifesto" class="navbar-item">Manifesto</NuxtLink>
        <NuxtLink to="/docs" class="navbar-item">Documentation</NuxtLink>
        <NuxtLink to="/essays" class="navbar-item">Essays</NuxtLink>
        <!-- Future: add /philosophy /essays -->
      </div>
    </div>
  </nav>
</template>

<script setup>
import { ref } from 'vue'

const isNavbarActive = ref(false)

function toggleNavbar() {
  isNavbarActive.value = !isNavbarActive.value
}
</script>

<style scoped>

.navbar {
  border-bottom: 1px solid #eaeaea;
}

.navbar-item {
  font-weight: 500;
}

.logo-text {
  font-weight: 600;
  font-size: 1.25rem;
}
</style>