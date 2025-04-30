<template>
  <div class="layout-container">
    <Navbar></Navbar>

    <!-- Main Content -->
    <section class="section doc-section">
      <div class="container doc-container">
        <slot />
      </div>
    </section>

    <SiteFooter></SiteFooter>
  </div>
</template>

<script setup>
</script>

<style scoped>
.layout-container {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.doc-section {
  flex: 1;
  padding-top: 3rem;
  padding-bottom: 3rem;
}

.doc-container {
  max-width: 740px;
  margin: 0 auto;
  padding: 0 1rem;
}

footer {
  padding: 2rem 1.5rem 2rem;
}

h1, h2, h3, h4, h5, h6 {
  font-weight: 600;
  margin-top: 2rem;
  margin-bottom: 1rem;
}

p {
  margin-bottom: 1.25rem;
  line-height: 1.7;
}

ul {
  margin-left: 1.5rem;
  list-style-type: disc;
}

a {
  color: #3273dc; /* Bulma primary color */
  text-decoration: underline;
}
</style>
