<script lang="ts" setup>
import type { VbenFormSchema } from '@vben/common-ui';
import type { Recordable } from '@vben/types';

import { computed, ref, reactive } from 'vue';
import { message } from 'ant-design-vue';
import { useRouter } from 'vue-router';

import { AuthenticationForgetPassword, z } from '@vben/common-ui';
import { sendEmailCodeApi, resetPasswordApi } from '#/api/core/auth';
import { $t } from '#/locales';

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
          placeholder: $t('page.auth.email'),
          type: 'email',
          size: 'large',
        },
        fieldName: 'email',
        label: $t('page.auth.email'),
        rules: z.string().email({ message: $t('page.auth.emailFormatInvalid') }),
      },
    ];
  } else {
    // 步骤2：验证码和新密码
    return [
      {
        component: 'VbenInput',
        componentProps: {
          placeholder: $t('page.auth.verificationCode'),
          size: 'large',
          maxlength: 6,
        },
        fieldName: 'code',
        label: $t('page.auth.verificationCode'),
        rules: z.string().min(6, { message: $t('page.auth.verificationCodeRequired') }).max(6, { message: $t('page.auth.verificationCodeLength') }),
      },
      {
        component: 'VbenInputPassword',
        componentProps: {
          passwordStrength: true,
          placeholder: $t('page.auth.enterNewPassword'),
          size: 'large',
        },
        fieldName: 'newPassword',
        label: $t('page.auth.newPassword'),
        renderComponentContent() {
          return {
            strengthText: () => $t('page.auth.passwordStrength'),
          };
        },
        rules: z.string().min(8, { message: $t('page.auth.passwordMinimumEight') }),
      },
    ];
  }
});

// 发送验证码
const sendEmailCode = async (email: string) => {
  sendingCode.value = true;
  try {
    const response = await sendEmailCodeApi({ email });
    message.success($t('page.auth.verificationCodeSent'));
    
    // 开发环境可能会返回验证码，便于测试
    if (import.meta.env.DEV && response.code) {
      console.log('验证码:', response.code);
      message.info(`${$t('page.auth.verificationCode')}: ${response.code}`);
    }
  } catch (error: any) {
    message.error(error.message || $t('page.auth.sendCodeFailed'));
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
      message.success($t('page.auth.verificationCodeSentNotice'));
    } else {
      // 步骤2：重置密码
      await resetPasswordApi({
        email: resetData.email,
        code: values.code,
        newPassword: values.newPassword,
      });
      
      message.success($t('page.auth.passwordResetSuccess'));
      
      // 重置成功后跳转到登录页面
      setTimeout(() => {
        router.push('/auth/login');
      }, 2000);
    }
  } catch (error: any) {
    message.error(error.message || $t('page.auth.operationFailed'));
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
  return currentStep.value === 1 ? $t('page.auth.resetPassword') : $t('page.auth.setNewPassword');
});

const stepSubtitle = computed(() => {
  return currentStep.value === 1 
    ? $t('page.auth.enterEmailToSendCode')
    : $t('page.auth.codeSentTo', { email: resetData.email });
});

const buttonText = computed(() => {
  return currentStep.value === 1 ? $t('page.auth.sendVerificationCode') : $t('page.auth.resetPassword');
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
            {{ $t('page.auth.changeEmail') }}
          </button>
        </div>
      </div>
      
      <!-- 密码设置提示（步骤2） -->
      <div v-if="currentStep === 2" class="mb-6 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
        <p class="text-sm font-medium text-[var(--text-primary)] mb-2">{{ $t('page.auth.newPasswordRequirements') }}</p>
        <ul class="space-y-1 text-sm text-[var(--text-secondary)]">
          <li class="flex items-center space-x-2">
            <svg class="w-4 h-4 text-green-500" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/>
            </svg>
            <span>{{ $t('page.auth.passwordMinimumEight') }}</span>
          </li>
        </ul>
      </div>
    </template>
  </AuthenticationForgetPassword>
</template>
