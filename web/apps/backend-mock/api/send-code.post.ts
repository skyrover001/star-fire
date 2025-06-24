export default defineEventHandler(async (event) => {
  const body = await readBody(event);
  const { email } = body;

  if (!email) {
    throw createError({
      statusCode: 400,
      statusMessage: '邮箱地址不能为空',
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

  // 模拟检查邮箱是否已注册
  const registeredEmails = ['admin@example.com', 'test@example.com'];
  if (registeredEmails.includes(email.toLowerCase())) {
    throw createError({
      statusCode: 400,
      statusMessage: '该邮箱已被注册，请使用其他邮箱或直接登录',
    });
  }

  // 模拟验证码（开发环境）
  const verificationCode = Math.floor(100000 + Math.random() * 900000).toString();

  // 模拟发送邮件的延迟
  await new Promise(resolve => setTimeout(resolve, 1200));

  console.log(`🚀 发送验证码到 ${email}: ${verificationCode}`);

  return {
    message: '验证码已发送到您的邮箱，请查收',
    code: verificationCode, // 开发环境返回验证码便于测试
    expiresIn: 300, // 5分钟过期
  };
});
