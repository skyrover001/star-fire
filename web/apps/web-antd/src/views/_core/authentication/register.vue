<script lang="ts" setup>
import type { VbenFormSchema } from '@vben/common-ui';
import type { Recordable } from '@vben/types';

import { computed, ref, reactive } from 'vue';
import { message, Modal as AModal, Button as AButton } from 'ant-design-vue';
import { useRouter } from 'vue-router';

import { AuthenticationRegister, z } from '@vben/common-ui';
import { sendEmailCodeApi, registerApi } from '#/api/core/auth';
import { $t } from '#/locales';

defineOptions({ name: 'Register' });

const router = useRouter();
const loading = ref(false);
const sendingCode = ref(false);
const agreeTerms = ref(false);
const agreePrivacy = ref(false);
const showDisclaimer = ref(false);
const showPrivacy = ref(false);
const registerFormRef = ref<InstanceType<typeof AuthenticationRegister>>();

// 从协议弹窗点击“同意并继续”时，同步勾选表单中的复选框
const agreeFromModal = () => {
  agreeTerms.value = true;
  showDisclaimer.value = false;
};

// 从隐私政策弹窗点击“同意并继续”时，同步勾选表单中的复选框
const agreePrivacyFromModal = () => {
  agreePrivacy.value = true;
  showPrivacy.value = false;
};

// 注册步骤状态：1-邮箱 2-密码和验证码
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
            placeholder: $t('page.auth.email'),
            type: 'email',
            class: 'h-12',
          },
          fieldName: 'email',
          label: '',
          rules: z.string().email({ message: $t('page.auth.emailFormatInvalid') }),
        },
      ];
    case 2:
      // 步骤2：密码和验证码
      return [
        {
          component: 'VbenInputPassword',
          componentProps: {
            passwordStrength: true,
            placeholder: $t('page.auth.password'),
            class: 'h-12',
          },
          fieldName: 'password',
          label: '',
          renderComponentContent() {
            return {
              strengthText: () => $t('page.auth.passwordRequirements'),
            };
          },
          rules: z.string().min(8, { message: $t('page.auth.passwordMinimumEight') }),
        },
        {
          component: 'VbenInputPassword',
          componentProps: {
            placeholder: $t('page.auth.confirmPassword'),
            class: 'h-12',
          },
          dependencies: {
            rules(values) {
              const { password } = values;
              return z
                .string({ required_error: $t('page.auth.confirmPasswordRequired') })
                .min(1, { message: $t('page.auth.confirmPasswordRequired') })
                .refine((value) => value === password, {
                  message: $t('page.auth.passwordMismatch'),
                });
            },
            triggerFields: ['password'],
          },
          fieldName: 'confirmPassword',
          label: '',
        },
        {
          component: 'VbenInput',
          componentProps: {
            placeholder: $t('page.auth.verificationCode'),
            class: 'h-12',
            maxlength: 6,
          },
          fieldName: 'emailCode',
          label: '',
          rules: z.string().min(6, { message: $t('page.auth.verificationCodeRequired') }).max(6, { message: $t('page.auth.verificationCodeLength') }),
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
      return $t('page.auth.createAccount');
    case 2:
      return $t('page.auth.createAccount');
    default:
      return $t('page.auth.createAccount');
  }
});

// 获取当前步骤的副标题
const stepSubtitle = computed(() => {
  switch (currentStep.value) {
    case 1:
      return '';
    case 2:
      return $t('page.auth.setPasswordAndEnterCode');
    default:
      return '';
  }
});

// 获取按钮文本
const buttonText = computed(() => {
  return $t('page.auth.continue');
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
      if (!agreeTerms.value || !agreePrivacy.value) {
        message.warning($t('page.auth.agreeTermsRequired'));
        return;
      }
      // 步骤1：保存邮箱，发送验证码，进入密码和验证码设置
      registrationData.email = values.email;
      await sendEmailCode(registrationData.email);
      currentStep.value = 2;
      message.success($t('page.auth.verificationCodeSentSetPassword'));
    } else if (currentStep.value === 2) {
      // 步骤2：验证码验证，完成注册
      if (!agreeTerms.value || !agreePrivacy.value) {
        message.warning($t('page.auth.agreeTermsRequired'));
        return;
      }
      await completeRegistration(values);
    }
  } catch (error: any) {
    // 发送验证码失败，保持在当前步骤
    message.error(error.message || $t('page.auth.operationFailed'));
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
      email: registrationData.email,
      password: values.password,
      code: values.emailCode,
    });
    
    message.success($t('page.auth.accountCreated'));
    console.log('register success:', response);
    
    // 注册成功后跳转到登录页面
    setTimeout(() => {
      router.push('/auth/login');
    }, 2000);
  } catch (error: any) {
    message.error(error.message || $t('page.auth.registrationFailed'));
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
  <div>
    <AuthenticationRegister
    ref="registerFormRef"
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
      <!-- 显示已输入的邮箱（步骤2） -->
      <div v-if="currentStep === 2" class="mb-6 p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
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
            {{ $t('page.auth.edit') }}
          </button>
        </div>
      </div>
      
      <!-- 密码强度提示（步骤2） -->
      <div v-if="currentStep === 2" class="mb-6 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
        <p class="text-sm font-medium text-[var(--text-primary)] mb-2">{{ $t('page.auth.passwordRequirementsTitle') }}</p>
        <ul class="space-y-1 text-sm text-[var(--text-secondary)]">
          <li class="flex items-center space-x-2">
            <svg class="w-4 h-4 text-green-500" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/>
            </svg>
            <span>{{ $t('page.auth.passwordMinimumEight') }}</span>
          </li>
          <li class="flex items-center space-x-2">
            <svg class="w-4 h-4 text-green-500" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/>
            </svg>
            <span>{{ $t('page.auth.confirmPasswordMatches') }}</span>
          </li>
        </ul>
        
        <!-- 验证码提示 -->
        <div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
          <p class="text-sm font-medium text-[var(--text-primary)] mb-2">{{ $t('page.auth.verificationCodeSentNotice') }}</p>
          <p class="text-sm text-[var(--text-secondary)]">{{ $t('page.auth.enterVerificationCodeNotice') }}</p>
        </div>
      </div>
    </template>

    <!-- 协议与隐私政策勾选 -->
    <template #extra>
      <div class="mt-4 space-y-3 text-sm">
        <label class="flex cursor-pointer items-start gap-2">
          <input
            v-model="agreeTerms"
            type="checkbox"
            class="mt-0.5 size-4 shrink-0 accent-blue-600"
          />
          <span class="text-[var(--text-secondary)]">
            {{ $t('page.auth.agreeTerms') }}
            <a
              class="text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-200 underline"
              href="javascript:void(0)"
              @click.prevent="showDisclaimer = true"
            >
              {{ $t('page.auth.viewTerms') }}
            </a>
          </span>
        </label>
        <label class="flex cursor-pointer items-start gap-2">
          <input
            v-model="agreePrivacy"
            type="checkbox"
            class="mt-0.5 size-4 shrink-0 accent-blue-600"
          />
          <span class="text-[var(--text-secondary)]">
            {{ $t('page.auth.agreePrivacy') }}
            <a
              class="text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-200 underline"
              href="javascript:void(0)"
              @click.prevent="showPrivacy = true"
            >
              {{ $t('page.auth.viewPrivacy') }}
            </a>
          </span>
        </label>
      </div>
    </template>
    
    <!-- 底部登录链接 -->
    <template #footer>
      <div class="text-center mt-6">
        <span class="text-[var(--text-secondary)]">{{ $t('page.auth.haveAccount') }}</span>
        <RouterLink
          :to="{ name: 'Login' }"
          class="text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-200 ml-1 font-medium"
        >
          {{ $t('page.auth.signIn') }}
        </RouterLink>
      </div>
    </template>
    </AuthenticationRegister>

    <!-- 免责声明与用户协议弹窗 -->
    <a-modal
    :open="showDisclaimer"
    :title="$t('page.auth.disclaimerTitle')"
    :footer="null"
    width="720px"
    @cancel="showDisclaimer = false"
  >
    <div class="max-h-[60vh] overflow-y-auto pr-2 text-sm leading-relaxed text-[var(--text-secondary)]">
      <p class="mb-4">{{ $t('page.auth.disclaimerSections.intro') }}</p>

      <h3 class="mb-2 mt-4 font-semibold text-[var(--text-primary)]">
        {{ $t('page.auth.disclaimerSections.account.title') }}
      </h3>
      <p class="whitespace-pre-line">{{ $t('page.auth.disclaimerSections.account.content') }}</p>

      <h3 class="mb-2 mt-4 font-semibold text-[var(--text-primary)]">
        {{ $t('page.auth.disclaimerSections.modelContributor.title') }}
      </h3>
      <p class="whitespace-pre-line">{{ $t('page.auth.disclaimerSections.modelContributor.content') }}</p>

      <h3 class="mb-2 mt-4 font-semibold text-[var(--text-primary)]">
        {{ $t('page.auth.disclaimerSections.platformDisclaimer.title') }}
      </h3>
      <p class="whitespace-pre-line">{{ $t('page.auth.disclaimerSections.platformDisclaimer.content') }}</p>

      <h3 class="mb-2 mt-4 font-semibold text-[var(--text-primary)]">
        {{ $t('page.auth.disclaimerSections.apiUsage.title') }}
      </h3>
      <p class="whitespace-pre-line">{{ $t('page.auth.disclaimerSections.apiUsage.content') }}</p>

      <p class="mt-4 font-medium text-[var(--text-primary)]">
        {{ $t('page.auth.disclaimerSections.acceptance') }}
      </p>
    </div>
    <div class="mt-6 flex justify-end gap-3">
      <a-button @click="showDisclaimer = false">
        {{ $t('page.auth.disclaimerClose') }}
      </a-button>
      <a-button
        type="primary"
        @click="agreeFromModal"
      >
        {{ $t('page.auth.disclaimerAgree') }}
      </a-button>
    </div>
    </a-modal>

    <!-- 隐私政策弹窗 -->
    <a-modal
    :open="showPrivacy"
    :title="$t('page.auth.privacyTitle')"
    :footer="null"
    width="720px"
    @cancel="showPrivacy = false"
  >
    <div class="max-h-[60vh] overflow-y-auto pr-2 text-sm leading-relaxed text-[var(--text-secondary)]">
      <p class="mb-4">{{ $t('page.auth.privacySections.intro') }}</p>

      <h3 class="mb-2 mt-4 font-semibold text-[var(--text-primary)]">
        {{ $t('page.auth.privacySections.collection.title') }}
      </h3>
      <p class="whitespace-pre-line">{{ $t('page.auth.privacySections.collection.content') }}</p>

      <h3 class="mb-2 mt-4 font-semibold text-[var(--text-primary)]">
        {{ $t('page.auth.privacySections.dataUsage.title') }}
      </h3>
      <p class="whitespace-pre-line">{{ $t('page.auth.privacySections.dataUsage.content') }}</p>

      <h3 class="mb-2 mt-4 font-semibold text-[var(--text-primary)]">
        {{ $t('page.auth.privacySections.sharing.title') }}
      </h3>
      <p class="whitespace-pre-line">{{ $t('page.auth.privacySections.sharing.content') }}</p>

      <h3 class="mb-2 mt-4 font-semibold text-[var(--text-primary)]">
        {{ $t('page.auth.privacySections.security.title') }}
      </h3>
      <p class="whitespace-pre-line">{{ $t('page.auth.privacySections.security.content') }}</p>

      <h3 class="mb-2 mt-4 font-semibold text-[var(--text-primary)]">
        {{ $t('page.auth.privacySections.userRights.title') }}
      </h3>
      <p class="whitespace-pre-line">{{ $t('page.auth.privacySections.userRights.content') }}</p>

      <p class="mt-4 font-medium text-[var(--text-primary)]">
        {{ $t('page.auth.privacySections.acceptance') }}
      </p>
    </div>
    <div class="mt-6 flex justify-end gap-3">
      <a-button @click="showPrivacy = false">
        {{ $t('page.auth.disclaimerClose') }}
      </a-button>
      <a-button
        type="primary"
        @click="agreePrivacyFromModal"
      >
        {{ $t('page.auth.disclaimerAgree') }}
      </a-button>
    </div>
    </a-modal>
  </div>
</template>
