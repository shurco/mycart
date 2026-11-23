<script lang="ts">
  import type { Product } from '$lib/types/models'
  import { getProductImageUrl } from '$lib/utils/imageUrl'
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    images: Product['images']
    name: string
  }

  let { images, name }: Props = $props()

  let currentSlide = $state(0)

  function nextSlide(length: number) {
    currentSlide = (currentSlide + 1) % length
  }

  function prevSlide(length: number) {
    currentSlide = (currentSlide + length - 1) % length
  }
</script>

<div>
  {#if !images || images.length === 0}
    <div class="relative h-[400px] border-4 border-black bg-white sm:h-[500px]">
      <img src="/assets/img/noimage.png" alt="" class="absolute inset-0 h-full w-full object-cover" />
    </div>
  {:else if images.length === 1}
    <div class="relative h-[400px] overflow-hidden border-4 border-black bg-white sm:h-[500px]">
      <img
        src={getProductImageUrl(images[0], 'md')}
        alt={name}
        class="absolute inset-0 h-full w-full object-cover"
      />
    </div>
  {:else}
    <div class="relative h-[400px] overflow-hidden border-4 border-black bg-white sm:h-[500px]">
      <div
        class="flex h-full w-full transition-transform duration-500 ease-in-out"
        style="transform: translateX(-{currentSlide * 100}%)"
      >
        {#each images as image (image.id || image.name)}
          <div class="h-full w-full flex-shrink-0">
            <img
              src={getProductImageUrl(image, 'md')}
              alt={name}
              class="block h-full w-full object-cover"
            />
          </div>
        {/each}
      </div>
      <button
        onclick={() => prevSlide(images.length)}
        class="absolute top-1/2 left-4 cursor-pointer border-4 border-black bg-yellow-300 p-3 text-xl font-black text-black transition-all duration-200 hover:-translate-x-1 hover:-translate-y-1 hover:shadow-[6px_6px_0px_0px_rgba(0,0,0,1)]"
        aria-label={t('product.previousImage')}
      >
        ←
      </button>
      <button
        onclick={() => nextSlide(images.length)}
        class="absolute top-1/2 right-4 cursor-pointer border-4 border-black bg-yellow-300 p-3 text-xl font-black text-black transition-all duration-200 hover:-translate-x-1 hover:-translate-y-1 hover:shadow-[6px_6px_0px_0px_rgba(0,0,0,1)]"
        aria-label={t('product.nextImage')}
      >
        →
      </button>
    </div>
  {/if}
</div>
