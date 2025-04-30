<template>
  <div class="layout-container">
    <!-- Navbar -->
    <Navbar></Navbar>

    <!-- Main Content -->
    <section class="section main-section">
      <div class="container content-container">
        <div class="columns">
          <div class="column is-4">
            <Sidebar></Sidebar>
          </div>
          <div class="column">
            <slot />
          </div>
        </div>
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

.main-section {
  flex: 1;
  padding-top: 2rem;
  padding-bottom: 2rem;
}

.content-container {
  max-width: 800px;
  margin: 0 auto;
}

.footer {
  padding: 2rem 1.5rem 2rem;
}
</style>
