<template>
  <svg viewBox="0 0 1024 1024" version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" v-bind="$attrs">
    <defs>
        <linearGradient id="bg_gradient" x1="0%" y1="0%" x2="0%" y2="100%">
            <stop offset="0%" style="stop-color:#FFFFFF;stop-opacity:1" />
            <stop offset="100%" style="stop-color:#F0F0F0;stop-opacity:1" />
        </linearGradient>

        <filter id="icon_shadow" x="-20%" y="-20%" width="140%" height="140%">
            <feGaussianBlur in="SourceAlpha" stdDeviation="20" /> <feOffset dx="0" dy="15" result="offsetblur" /> <feComponentTransfer>
            <feFuncA type="linear" slope="0.3" /> </feComponentTransfer>
            <feMerge>
                <feMergeNode />
                <feMergeNode in="SourceGraphic" /> </feMerge>
        </filter>
    </defs>

    <g transform="translate(128, 128) scale(0.75)" filter="url(#icon_shadow)">
        <path d="M128 128h768v768H128z" fill="#7dc5eb"></path>
        <path d="M256 256h512v512H256V256zM0 256h128v96H0V256z m0 192h128v96H0v-96z m0 192h128v96H0v-96zM896 256h128v96h-128V256z m0 192h128v96h-128v-96z m0 192h128v96h-128v-96zM768 0v128h-96V0h96z m-192 0v128h-96V0h96z m-192 0v128H288V0h96z m352 896v128h-96v-128h96z m-192 0v128h-96v-128h96z m-192 0v128H256v-128h96z"
              fill="#333333"></path>
    </g>
  </svg>
</template>
