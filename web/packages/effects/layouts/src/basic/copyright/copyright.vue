<script lang="ts" setup>
interface Props {
  companyName?: string;
  companySiteLink?: string;
  date?: string;
  icp?: string;
  icpLink?: string;
}

defineOptions({
  name: 'Copyright',
});

const props = withDefaults(defineProps<Props>(), {
  companyName: 'Star Fire',
  companySiteLink: '',
  date: new Date().getFullYear().toString(),
  icp: '',
  icpLink: '',
});
</script>

<template>
  <div class="text-md flex-center">
    <!-- ICP Link -->
    <a
      v-if="props.icp"
      :href="props.icpLink || 'javascript:void(0)'"
      class="hover:text-primary-hover mx-1"
      target="_blank"
    >
      {{ props.icp }}
    </a>

    <!-- Copyright Text -->
    <!-- Copyright © {{ date }} -->

     Copyright © {{ props.date }}

    <!-- Company Link -->
    <a
      v-if="props.companyName"
      :href="props.companySiteLink || 'javascript:void(0)'"
      class="hover:text-primary-hover mx-1"
      target="_blank"
    >
      <!-- {{ companyName }} -->
        TH-Star-Fire
    </a>
  </div>
</template>
