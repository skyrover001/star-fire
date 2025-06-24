import {
  clearRefreshTokenCookie,
  setRefreshTokenCookie,
} from '~/utils/cookie-utils';
import { generateAccessToken, generateRefreshToken } from '~/utils/jwt-utils';
import { forbiddenResponse } from '~/utils/response';

export default defineEventHandler(async (event) => {
  const { password, username } = await readBody(event);
  if (!password || !username) {
    setResponseStatus(event, 400);
    return useResponseError(
      'BadRequestException',
      'Username and password are required',
    );
  }

  // 判断输入的是邮箱还是用户名，支持两种方式登录
  const isEmail = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(username);
  
  const findUser = MOCK_USERS.find((item) => {
    const matchesCredential = isEmail 
      ? item.email === username 
      : item.username === username;
    return matchesCredential && item.password === password;
  });

  if (!findUser) {
    clearRefreshTokenCookie(event);
    return forbiddenResponse(event, 'Username or password is incorrect.');
  }

  const accessToken = generateAccessToken(findUser);
  const refreshToken = generateRefreshToken(findUser);

  setRefreshTokenCookie(event, refreshToken);

  return useResponseSuccess({
    ...findUser,
    accessToken,
  });
});
