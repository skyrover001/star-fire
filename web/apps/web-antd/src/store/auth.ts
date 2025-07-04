import type { Recordable, UserInfo } from '@vben/types';

import { ref } from 'vue';

import { LOGIN_PATH } from '@vben/constants';
import { preferences } from '@vben/preferences';
import { resetAllStores, useAccessStore, useUserStore } from '@vben/stores';

import { notification } from 'ant-design-vue';
import { defineStore } from 'pinia';

import { loginApi, logoutApi, getUserInfoApi } from '#/api';
import { $t } from '#/locales';
import { router } from '#/router';

export const useAuthStore = defineStore('auth', () => {
  const accessStore = useAccessStore();
  const userStore = useUserStore();

  const loginLoading = ref(false);

  /**
   * 异步处理登录操作
   * Asynchronously handle the login process
   * @param params 登录表单数据
   */
  async function authLogin(
    params: Recordable<any>,
    onSuccess?: () => Promise<void> | void,
  ) {
    // 异步处理用户登录操作并获取 accessToken
    let userInfo: null | UserInfo = null;
    try {
      loginLoading.value = true;
      const { token, user, expires_in } = await loginApi(params as any);
      console.log('authLogin', params, 'token:', token, 'user:', user, 'expires_in:', expires_in);

      // 如果成功获取到 accessToken
      if (token) {
        console.log('登录成功','token:', token, 'user:', user, 'expires_in:', expires_in);
        accessStore.setAccessToken(token);

        // 获取用户信息并存储到 accessStore 中
        // const [fetchUserInfoResult, accessCodes] = await Promise.all([
        //   fetchUserInfo(),
        //   getAccessCodesApi(),
        // ]);
        // userInfo = fetchUserInfoResult;

        // 转换用户信息格式以匹配 UserInfo 接口
        userInfo = {
          userId: user.id,
          username: user.username,
          realName: user.username,
          avatar: '',
          roles: [user.role],
          desc: '',
          homePath: preferences.app.defaultHomePath,
          token: token,
          email: user.email,
        } as UserInfo;
        
        userStore.userInfo = userInfo;
        userStore.userRoles = [user.role];
        accessStore.setLoginExpired(false);
        // 设置用户权限码
        accessStore.setAccessCodes([]);

        // const accessCodes = await getAccessCodesApi();
        // accessStore.setAccessCodes(accessCodes);
     

        if (accessStore.loginExpired) {
          accessStore.setLoginExpired(false);
        } else {
          onSuccess
            ? await onSuccess?.()
            : await router.push(
                userInfo?.homePath || preferences.app.defaultHomePath,
              );
        }

        if (userInfo?.realName) {
          notification.success({
            description: `${$t('authentication.loginSuccessDesc')}:${userInfo?.realName}`,
            duration: 3,
            message: $t('authentication.loginSuccess'),
          });
        }
      }
    } finally {
      loginLoading.value = false;
    }

    return {
      userInfo,
    };
  }

  async function logout(redirect: boolean = true) {
    try {
      await logoutApi();
    } catch {
      // 不做任何处理
    }
    resetAllStores();
    accessStore.setLoginExpired(false);

    // 回登录页带上当前路由地址
    await router.replace({
      path: LOGIN_PATH,
      query: redirect
        ? {
            redirect: encodeURIComponent(router.currentRoute.value.fullPath),
          }
        : {},
    });
  }

  async function fetchUserInfo() {
    try {
      // 如果用户信息已经存在，直接返回
      if (userStore.userInfo) {
        return userStore.userInfo;
      }
      
      const apiUserInfo = await getUserInfoApi();
      
      // 直接使用API返回的用户信息，因为它已经符合UserInfo接口
      userStore.userInfo = apiUserInfo;
      userStore.userRoles = apiUserInfo.roles || [];
      return apiUserInfo;
    } catch (error) {
      console.error('获取用户信息失败:', error);
      // 如果获取失败，清除用户信息
      userStore.userInfo = null;
      userStore.userRoles = [];
      return null;
    }
  }

  function $reset() {
    loginLoading.value = false;
  }

  return {
    $reset,
    authLogin,
    fetchUserInfo,
    loginLoading,
    logout,
  };
});
