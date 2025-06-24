import { defineOverridesPreferences } from '@vben/preferences';

/**
 * @description 项目配置文件
 * 只需要覆盖项目中的一部分配置，不需要的配置不用覆盖，会自动使用默认配置
 * !!! 更改配置后请清空缓存，否则可能不生效
 */
export const overridesPreferences = defineOverridesPreferences({
  // overrides
  app: {
    name: import.meta.env.VITE_APP_TITLE,
  },
  // 自定义Logo配置
  logo: {
    enable: true,
    fit: 'contain', // 'contain' | 'cover' | 'fill' | 'none' | 'scale-down'
    source: '/logo.png', // 自定义logo文件路径，请将您的PNG文件放在 public/logo.png
  },
});
