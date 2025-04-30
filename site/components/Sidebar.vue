<template>
  <aside class="menu sidebar">
    <p class="menu-label">
      BlockParty
    </p>
    <ul class="menu-list">
      <li>
        <NuxtLink to="/" exact-active-class="is-active">Home</NuxtLink>
      </li>
      <li>
        <NuxtLink to="/manifesto" exact-active-class="is-active">Manifesto</NuxtLink>
      </li>
      <li>
        <NuxtLink to="/whitepaper" exact-active-class="is-active">Whitepaper</NuxtLink>
      </li>
      <li>
        <details>
          <summary>Essays</summary>
          <ul>
            <li><NuxtLink to="/essays/why-we-built-it" exact-active-class="is-active">Why We Built BlockParty</NuxtLink></li>
            <li><NuxtLink to="/essays/post-network" exact-active-class="is-active">Post-Network Future</NuxtLink></li>
            <li><NuxtLink to="/essays/not-just-a-file-format" exact-active-class="is-active">Not Just a File Format</NuxtLink></li>
            <li><NuxtLink to="/essays/internet-isnt-forever" exact-active-class="is-active">Internet Isn’t Forever</NuxtLink></li>
            <li><NuxtLink to="/essays/expiring-trust-enduring-humanity" exact-active-class="is-active">Expiring Trust</NuxtLink></li>
            <li><NuxtLink to="/essays/burn-it-all" exact-active-class="is-active">Burn It All</NuxtLink></li>
            <li><NuxtLink to="/essays/a-signal-you-cant-stop" exact-active-class="is-active">A Signal You Can’t Stop</NuxtLink></li>
            <li><NuxtLink to="/essays/100-ways-to-carry-a-block" exact-active-class="is-active">100 Ways to Carry a Block</NuxtLink></li>
          </ul>
        </details>
      </li>
      <li>
        <NuxtLink to="/docs" exact-active-class="is-active">Docs</NuxtLink>
      </li>
    </ul>
  </aside>
</template>

<script setup>
// No special logic needed for now — purely declarative sidebar
</script>

<style scoped>
.sidebar {
  position: sticky;
  top: 1rem;
  max-height: calc(100vh - 2rem);
  overflow-y: auto;
  padding-right: 1rem;
  min-width: 230px;
}

.menu-list li {
  margin-bottom: 0.5rem;
}

.menu-list a {
  display: block;
  padding: 0.5rem 0.75rem;
  color: #ccc;
  text-decoration: none;
}

.menu-list a:hover {
  color: #fff;
}

.is-active {
  font-weight: bold;
  color: #7db2ff;
}
</style>
