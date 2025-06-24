<script lang="ts" setup>
import type { VbenFormSchema } from '@vben/common-ui';
import type { Recordable } from '@vben/types';

import { computed, ref, reactive } from 'vue';
import { message } from 'ant-design-vue';
import { useRouter } from 'vue-router';

import { AuthenticationForgetPassword, z } from '@vben/common-ui';
import { sendEmailCodeApi, resetPasswordApi } from '#/api/core/auth';

defineOptions({ name: 'ForgetPassword' });

const router = useRouter();
const loading = ref(false);
const sendingCode = ref(false);

// 重置步骤：1-发送验证码 2-重置密码
const currentStep = ref(1);
const resetData = reactive({
  email: '',
  code: '',
  newPassword: '',
});

// 当前步骤的表单配置
const formSchema = computed((): VbenFormSchema[] => {
  if (currentStep.value === 1) {
    // 步骤1：邮箱验证
    return [
      {
        component: 'VbenInput',
        componentProps: {
          placeholder: '请输入您的邮箱地址',
          type: 'email',
          size: 'large',
        },
        fieldName: 'email',
        label: '邮箱地址',
        rules: z.string().email({ message: '请输入正确的邮箱格式' }),
      },
    ];
  } else {
    // 步骤2：验证码和新密码
    return [
      {
        component: 'VbenInput',
        componentProps: {
          placeholder: '请输入6位验证码',
          size: 'large',
          maxlength: 6,
        },
        fieldName: 'code',
        label: '验证码',
        rules: z.string().min(6, { message: '请输入6位验证码' }).max(6, { message: '验证码为6位数字' }),
      },
      {
        component: 'VbenInputPassword',
        componentProps: {
          passwordStrength: true,
          placeholder: '请输入新密码',
          size: 'large',
        },
        fieldName: 'newPassword',
        label: '新密码',
        renderComponentContent() {
          return {
            strengthText: () => '密码强度',
          };
        },
        rules: z.string().min(8, { message: '密码至少8个字符' }),
      },
    ];
  }
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
      // 步骤1：发送验证码
      resetData.email = values.email;
      await sendEmailCode(resetData.email);
      currentStep.value = 2;
      message.success('验证码已发送，请查收邮件');
    } else {
      // 步骤2：重置密码
      await resetPasswordApi({
        email: resetData.email,
        code: values.code,
        newPassword: values.newPassword,
      });
      
      message.success('密码重置成功！即将跳转到登录页面');
      
      // 重置成功后跳转到登录页面
      setTimeout(() => {
        router.push('/auth/login');
      }, 2000);
    }
  } catch (error: any) {
    message.error(error.message || '操作失败，请稍后重试');
  } finally {
    loading.value = false;
  }
}

// 返回上一步
const goBack = () => {
  if (currentStep.value > 1) {
    currentStep.value = 1;
  }
};

// 获取当前步骤的标题和副标题
const stepTitle = computed(() => {
  return currentStep.value === 1 ? '重置密码' : '设置新密码';
});

const stepSubtitle = computed(() => {
  return currentStep.value === 1 
    ? '输入您的邮箱地址，我们将发送验证码'
    : `验证码已发送到 ${resetData.email}`;
});

const buttonText = computed(() => {
  return currentStep.value === 1 ? '发送验证码' : '重置密码';
});
</script>

<template>
  <AuthenticationForgetPassword
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
        <p class="text-[var(--text-secondary)] text-base">
          {{ stepSubtitle }}
        </p>
      </div>
    </template>
    
    <template #subTitle>
      <!-- 显示已输入的邮箱（步骤2） -->
      <div v-if="currentStep === 2" class="mb-6 p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
        <div class="flex items-center justify-between">
          <div class="flex items-center space-x-2">
            <svg class="w-4 h-4 text-blue-600 dark:text-blue-400" fill="currentColor" viewBox="0 0 20 20">
              <path d="M2.003 5.884L10 9.882l7.997-3.998A2 2 0 0016 4H4a2 2 0 00-1.997 1.884z"/>
              <path d="M18 8.118l-8 4-8-4V14a2 2 0 002 2h12a2 2 0 002-2V8.118z"/>
            </svg>
            <span class="text-sm font-medium text-blue-800 dark:text-blue-200">{{ resetData.email }}</span>
          </div>
          <button 
            @click="goBack"
            class="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-200"
            type="button"
          >
            更改邮箱
          </button>
        </div>
      </div>
      
      <!-- 密码设置提示（步骤2） -->
      <div v-if="currentStep === 2" class="mb-6 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
        <p class="text-sm font-medium text-[var(--text-primary)] mb-2">新密码要求:</p>
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
  </AuthenticationForgetPassword>
</template>
