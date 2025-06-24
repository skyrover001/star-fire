import { baseRequestClient, requestClient } from '#/api/request';

export interface UserInfo {
  id: string
  username: string
  email: string
  role: string
  created_at: string
  updated_at: string
}

export namespace AuthApi {
  /** 登录接口参数 */
  export interface LoginParams {
    username: string;  // 可以是用户名或邮箱
    password: string;
  }

  /** 登录接口返回值 */
  export interface LoginResult {
    token: string;
    user: UserInfo;
    expires_in: number;
  }

  export interface RefreshTokenResult {
    data: string;
    status: number;
  }

  /** 注册接口参数 */
  export interface RegisterParams {
    username: string;
    email: string;
    password: string;
    code: string;
  }

  /** 注册接口返回值 */
  export interface RegisterResult {
    message: string;
    user: UserInfo;
  }

  /** 发送验证码参数 */
  export interface SendCodeParams {
    email: string;
  }

  /** 发送验证码返回值 */
  export interface SendCodeResult {
    message: string;
    code?: string; // 开发环境可能返回验证码
  }

  /** 重置密码参数 */
  export interface ResetPasswordParams {
    email: string;
    code: string;
    newPassword: string;
  }

  /** 重置密码返回值 */
  export interface ResetPasswordResult {
    message: string;
  }
}

/**
 * 登录
 */
export async function loginApi(data: AuthApi.LoginParams) {
  try {
    const res = await requestClient.post<AuthApi.LoginResult>('/login', data)
    return res
  } catch (error: any) {
    throw error
  }
}

/**
 * 刷新accessToken
 */
export async function refreshTokenApi() {
  return baseRequestClient.post<AuthApi.RefreshTokenResult>('/auth/refresh', {
    withCredentials: true,
  });
}

/**
 * 退出登录
 */
export async function logoutApi() {
  return baseRequestClient.post('/auth/logout', {
    withCredentials: true,
  });
}

/**
 * 获取用户权限码
 */
export async function getAccessCodesApi() {
  return requestClient.get<string[]>('/auth/codes');
}

/**
 * 发送邮箱验证码
 */
export async function sendEmailCodeApi(data: AuthApi.SendCodeParams) {
  try {
    const res = await requestClient.post<AuthApi.SendCodeResult>('/send-code', data);
    return res;
  } catch (error: any) {
    throw error;
  }
}

/**
 * 用户注册
 */
export async function registerApi(data: AuthApi.RegisterParams) {
  try {
    const res = await requestClient.post<AuthApi.RegisterResult>('/register', data);
    return res;
  } catch (error: any) {
    throw error;
  }
}

/**
 * 重置密码
 */
export async function resetPasswordApi(data: AuthApi.ResetPasswordParams) {
  try {
    const res = await requestClient.post<AuthApi.ResetPasswordResult>('/reset-password', data);
    return res;
  } catch (error: any) {
    throw error;
  }
}
