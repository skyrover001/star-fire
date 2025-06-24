export default defineEventHandler(async (event) => {
  const body = await readBody(event);
  const { username, email, password, code } = body;

  // 参数验证
  if (!username) {
    throw createError({
      statusCode: 400,
      statusMessage: '用户名不能为空',
    });
  }

  if (username.length < 3) {
    throw createError({
      statusCode: 400,
      statusMessage: '用户名至少需要3个字符',
    });
  }

  if (!email) {
    throw createError({
      statusCode: 400,
      statusMessage: '邮箱地址不能为空',
    });
  }

  if (!code) {
    throw createError({
      statusCode: 400,
      statusMessage: '邮箱验证码不能为空',
    });
  }

  if (!password) {
    throw createError({
      statusCode: 400,
      statusMessage: '密码不能为空',
    });
  }

  if (password.length < 8) {
    throw createError({
      statusCode: 400,
      statusMessage: '密码至少需要8个字符',
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

  // 模拟验证码验证（在实际应用中，应该从缓存或数据库中验证）
  if (!/^\d{6}$/.test(code)) {
    throw createError({
      statusCode: 400,
      statusMessage: '验证码格式不正确，请输入6位数字',
    });
  }

  // 模拟检查用户名是否已存在
  const existingUsernames = ['admin', 'test', 'user', 'root'];
  if (existingUsernames.includes(username.toLowerCase())) {
    throw createError({
      statusCode: 400,
      statusMessage: '用户名已存在，请选择其他用户名',
    });
  }

  // 模拟检查邮箱是否已注册
  const registeredEmails = ['admin@example.com', 'test@example.com'];
  if (registeredEmails.includes(email.toLowerCase())) {
    throw createError({
      statusCode: 400,
      statusMessage: '该邮箱已被注册',
    });
  }

  // 模拟注册延迟
  await new Promise(resolve => setTimeout(resolve, 1800));

  // 模拟创建用户
  const newUser = {
    id: Math.random().toString(36).substr(2, 9),
    username,
    email,
    role: 'user',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  };

  console.log('🎉 新用户注册成功:', newUser);

  return {
    message: '恭喜您！账户创建成功',
    user: newUser,
  };
});
