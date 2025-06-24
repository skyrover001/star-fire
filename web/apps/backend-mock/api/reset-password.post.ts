export default defineEventHandler(async (event) => {
  const body = await readBody(event);
  const { email, code, newPassword } = body;

  // 参数验证
  if (!email) {
    throw createError({
      statusCode: 400,
      statusMessage: '邮箱地址不能为空',
    });
  }

  if (!code) {
    throw createError({
      statusCode: 400,
      statusMessage: '验证码不能为空',
    });
  }

  if (!newPassword) {
    throw createError({
      statusCode: 400,
      statusMessage: '新密码不能为空',
    });
  }

  // 验证邮箱格式
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  if (!emailRegex.test(email)) {
    throw createError({
      statusCode: 400,
      statusMessage: '请输入正确的邮箱格式',
    });
  }

  // 验证密码长度
  if (newPassword.length < 8) {
    throw createError({
      statusCode: 400,
      statusMessage: '密码至少需要8个字符',
    });
  }

  // 模拟验证码验证
  if (!/^\d{6}$/.test(code)) {
    throw createError({
      statusCode: 400,
      statusMessage: '验证码格式不正确，请输入6位数字',
    });
  }

  // 模拟检查邮箱是否存在（实际应用中需要从数据库查询）
  const existingEmails = ['vben@example.com', 'admin@example.com', 'jack@example.com'];
  if (!existingEmails.includes(email)) {
    throw createError({
      statusCode: 404,
      statusMessage: '该邮箱地址未注册',
    });
  }

  // 模拟验证码校验（实际应用中应该从缓存或数据库验证）
  // 这里简单模拟，实际开发中需要验证验证码是否正确且未过期
  const validCodes = ['123456', '888888'];
  if (!validCodes.includes(code)) {
    throw createError({
      statusCode: 400,
      statusMessage: '验证码错误或已过期',
    });
  }

  // 模拟更新密码成功
  // 实际应用中需要：
  // 1. 验证验证码
  // 2. 更新数据库中的用户密码
  // 3. 清除验证码缓存
  
  return {
    message: '密码重置成功，请使用新密码登录',
  };
});
