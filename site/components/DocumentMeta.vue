<template>
  <div class="doc-meta">
    <h1 class="title">{{ title }}</h1>

    <div class="doc-meta-links">
      <p class="canonical">
        <strong>Canonical:</strong>
        <a :href="canonical" target="_blank" rel="noopener noreferrer">{{ canonical }}</a>
      </p>

      <p class="source-note">
        <strong>Source:</strong>
        <a :href="source" target="_blank" rel="noopener noreferrer">
          BlockParty Protocol Repository
        </a>
      </p>
    </div>


    <div class="field is-grouped is-grouped-multiline">
      <div class="control">
        <div class="tags has-addons">
          <span class="tag is-dark">Version</span>
          <span class="tag is-info">{{ version }}</span>
        </div>
      </div>
      <div class="control">
        <div class="tags has-addons">
          <span class="tag is-dark">Last Updated</span>
          <span class="tag is-light">{{ updated }}</span>
        </div>
      </div>
      <div class="control">
        <div class="tags has-addons">
          <span class="tag is-dark">Status</span>
          <span class="tag is-success">{{ status }}</span>
        </div>
      </div>
      <div class="control">
        <div class="tags has-addons">
          <span class="tag is-dark">License</span>
          <span class="tag is-warning">{{ license }}</span>
        </div>
      </div>
    </div>
    <hr>
  </div>
</template>

<script setup>
defineProps({
  title: String,
  version: String,
  updated: String,
  status: String,
  source: String,
  license: String,
  canonical: String
})
</script>

<style scoped>
.doc-meta {
  margin-bottom: 2rem;
}

.meta-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  margin-top: 1rem;
  margin-bottom: 1.5rem;
}

.canonical {
  margin-top: 0.5rem;
  font-size: 0.9rem;
}
</style>
