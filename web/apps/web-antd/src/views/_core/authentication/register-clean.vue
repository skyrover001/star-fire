<script lang="ts" setup>
import type { VbenFormSchema } from '@vben/common-ui';
import type { Recordable } from '@vben/types';

import { computed, ref, reactive } from 'vue';
import { message } from 'ant-design-vue';
import { useRouter } from 'vue-router';

import { AuthenticationRegister, z } from '@vben/common-ui';
import { sendEmailCodeApi, registerApi } from '#/api/core/auth';

defineOptions({ name: 'Register' });

const router = useRouter();
const loading = ref(false);
const sendingCode = ref(false);

// 注册步骤状态：1-邮箱 2-密码 3-验证码
const currentStep = ref(1);
const registrationData = reactive({
  email: '',
  password: '',
  emailCode: '',
});

// 当前步骤的表单配置
const formSchema = computed((): VbenFormSchema[] => {
  switch (currentStep.value) {
    case 1:
      // 步骤1：邮箱输入
      return [
        {
          component: 'VbenInput',
          componentProps: {
            placeholder: '电子邮件地址',
            type: 'email',
            size: 'large',
            class: 'h-12',
          },
          fieldName: 'email',
          label: '',
          rules: z.string().email({ message: '请输入正确的邮箱格式' }),
        },
      ];
    case 2:
      // 步骤2：密码设置
      return [
        {
          component: 'VbenInputPassword',
          componentProps: {
            passwordStrength: true,
            placeholder: '密码',
            size: 'large',
            class: 'h-12',
          },
          fieldName: 'password',
          label: '',
          renderComponentContent() {
            return {
              strengthText: () => '您的密码必须包含:',
            };
          },
          rules: z.string().min(8, { message: '密码至少8个字符' }),
        },
      ];
    case 3:
      // 步骤3：验证码
      return [
        {
          component: 'VbenInput',
          componentProps: {
            placeholder: '代码',
            size: 'large',
            class: 'h-12',
            maxlength: 6,
          },
          fieldName: 'emailCode',
          label: '',
          rules: z.string().min(6, { message: '请输入6位验证码' }).max(6, { message: '验证码为6位数字' }),
        },
      ];
    default:
      return [];
  }
});

// 获取当前步骤的标题
const stepTitle = computed(() => {
  switch (currentStep.value) {
    case 1:
      return '创建帐户';
    case 2:
      return '创建帐户';
    case 3:
      return '检查您的收件箱';
    default:
      return '创建帐户';
  }
});

// 获取当前步骤的副标题
const stepSubtitle = computed(() => {
  switch (currentStep.value) {
    case 1:
      return '';
    case 2:
      return '设置 OpenAI 的密码以继续';
    case 3:
      return `输入我们刚向 ${registrationData.email} 发送的验证码。`;
    default:
      return '';
  }
});

// 获取按钮文本
const buttonText = computed(() => {
  return '继续';
});

// 发送验证码
const sendEmailCode = async (email: string) => {
  sendingCode.value = true;
  try {
    const response = await sendEmailCodeApi({ email });
    message.success('验证码已发送到您的邮箱');
    
    // 开发环境可能会返回验证码，便于测试
    if (import.meta.env.DEV && response.code) {
      console.log('验证码:', response.code);
      message.info(`验证码: ${response.code}`);
    }
  } catch (error: any) {
    message.error(error.message || '发送验证码失败，请稍后重试');
    throw error;
  } finally {
    sendingCode.value = false;
  }
};

// 处理表单提交
async function handleSubmit(values: Recordable<any>) {
  loading.value = true;
  
  try {
    if (currentStep.value === 1) {
      // 步骤1：保存邮箱，进入密码设置
      registrationData.email = values.email;
      currentStep.value = 2;
      message.success('邮箱验证通过');
    } else if (currentStep.value === 2) {
      // 步骤2：保存密码，发送验证码
      registrationData.password = values.password;
      await sendEmailCode(registrationData.email);
      currentStep.value = 3;
    } else if (currentStep.value === 3) {
      // 步骤3：验证码验证，完成注册
      await completeRegistration(values);
    }
  } catch (error: any) {
    // 发送验证码失败，回到步骤2
    if (currentStep.value === 3 && error.message?.includes('发送')) {
      currentStep.value = 2;
    }
  } finally {
    loading.value = false;
  }
}

// 完成注册
const completeRegistration = async (values: any) => {
  try {
    const emailParts = registrationData.email.split('@');
    const username = emailParts[0] || 'user'; // 使用邮箱前缀作为用户名
    
    const response = await registerApi({
      username,
      password: registrationData.password,
      confirmPassword: registrationData.password,
      email: registrationData.email,
      emailCode: values.emailCode,
      agreePolicy: true, // 默认同意
    });
    
    message.success('恭喜您！账户创建成功！');
    console.log('register success:', response);
    
    // 注册成功后跳转到登录页面
    setTimeout(() => {
      router.push('/auth/login');
    }, 2000);
  } catch (error: any) {
    message.error(error.message || '注册失败，请稍后重试');
    console.error('register error:', error);
    throw error;
  }
};

// 返回上一步
const goBack = () => {
  if (currentStep.value > 1) {
    currentStep.value = currentStep.value - 1;
  }
};
</script>

<template>
  <AuthenticationRegister
    :form-schema="formSchema"
    :loading="loading"
    :submit-button-text="buttonText"
    @submit="handleSubmit"
  >
    <template #title>
      <div class="text-center">
        <h1 class="text-3xl font-bold text-[var(--text-primary)] mb-2">
          {{ stepTitle }}
        </h1>
        <p v-if="stepSubtitle" class="text-[var(--text-secondary)] text-base">
          {{ stepSubtitle }}
        </p>
      </div>
    </template>
    
    <template #subTitle>
      <!-- 显示已输入的邮箱（步骤2和3） -->
      <div v-if="currentStep > 1" class="mb-6 p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
        <div class="flex items-center justify-between">
          <div class="flex items-center space-x-2">
            <svg class="w-4 h-4 text-blue-600 dark:text-blue-400" fill="currentColor" viewBox="0 0 20 20">
              <path d="M2.003 5.884L10 9.882l7.997-3.998A2 2 0 0016 4H4a2 2 0 00-1.997 1.884z"/>
              <path d="M18 8.118l-8 4-8-4V14a2 2 0 002 2h12a2 2 0 002-2V8.118z"/>
            </svg>
            <span class="text-sm font-medium text-blue-800 dark:text-blue-200">{{ registrationData.email }}</span>
          </div>
          <button 
            @click="goBack"
            class="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-200"
            type="button"
          >
            编辑
          </button>
        </div>
      </div>
      
      <!-- 密码强度提示（步骤2） -->
      <div v-if="currentStep === 2" class="mb-6 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
        <p class="text-sm font-medium text-[var(--text-primary)] mb-2">您的密码必须包含:</p>
        <ul class="space-y-1 text-sm text-[var(--text-secondary)]">
          <li class="flex items-center space-x-2">
            <svg class="w-4 h-4 text-green-500" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/>
            </svg>
            <span>至少 8 个字符</span>
          </li>
        </ul>
      </div>
    </template>
    
    <!-- 底部登录链接 -->
    <template #footer>
      <div class="text-center mt-6">
        <span class="text-[var(--text-secondary)]">已经有帐户了？</span>
        <a 
          href="/auth/login" 
          class="text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-200 ml-1 font-medium"
          @click.prevent="router.push('/auth/login')"
        >
          请登录
        </a>
        
        <div class="mt-4 text-center">
          <span class="text-[var(--text-tertiary)]">或</span>
        </div>
      </div>
    </template>
  </AuthenticationRegister>
</template>
