import json
import unittest
from unittest import mock

import i18n
import star_fire


class I18nTests(unittest.TestCase):
    def test_income_message_currency_is_normalized_to_rmb(self):
        self.assertEqual(
            star_fire.parse_income_message('income: 1.25 USD'),
            (True, '1.25', '¥'),
        )

    def test_calculate_income_record_supports_go_json_fields(self):
        item = {
            'ippm': 4, 'oppm': 8, 'cippm': 1,
            'input_tokens': 1000, 'output_tokens': 500, 'cached_tokens': 200,
        }
        self.assertAlmostEqual(star_fire.calculate_income_record(item), 0.0074)

    def test_calculate_income_record_supports_legacy_fields_and_caps_cached_tokens(self):
        item = {
            'IPPM': 4, 'OPPM': 8, 'CIPPM': 1,
            'InputTokens': 100, 'OutputTokens': 50, 'CachedTokens': 200,
        }
        self.assertAlmostEqual(star_fire.calculate_income_record(item), 0.0005)

    def test_start_requires_model_configuration_before_requesting_token(self):
        app = object.__new__(star_fire.StarFireAPP)
        app.user_stopped = True
        app.restart_attempt = 0
        app.restart_after_id = None
        app.host_entry = mock.Mock()
        app.host_entry.get.return_value = 'http://example.test'
        app.config = {'registered_models': []}
        app.get_register_token = mock.Mock()
        app._message = mock.Mock()

        app.start_starfire()

        app.get_register_token.assert_not_called()
        app._message.assert_called_once()
        self.assertEqual(app._message.call_args.args[1], '请先配置模型')

    def test_legacy_vllm_config_migrates_to_ollama(self):
        app = object.__new__(star_fire.StarFireAPP)
        app.config_file = 'unused.json'
        with mock.patch.object(star_fire.os.path, 'exists', return_value=False):
            app.load_config()
        app.config['model_mode'] = 'vllm'
        with mock.patch.object(star_fire.os.path, 'exists', return_value=True), \
             mock.patch('builtins.open', mock.mock_open(read_data=json.dumps({'model_mode': 'vllm'}))):
            app.load_config()
        self.assertEqual(app.config['model_mode'], 'ollama')

    def test_normalizes_supported_locale_variants(self):
        self.assertEqual(i18n.normalize_language("zh-CN"), "zh_CN")
        self.assertEqual(i18n.normalize_language("en_GB"), "en_US")
        self.assertIsNone(i18n.normalize_language("fr_FR"))

    def test_configured_language_overrides_detection(self):
        with mock.patch.object(i18n, "detect_system_language", return_value="zh_CN"):
            translator = i18n.I18n("en_US")

        self.assertEqual(translator.language, "en_US")
        self.assertEqual(translator("common.success"), "Success")

    def test_detected_language_is_used_without_override(self):
        with mock.patch.object(i18n, "detect_system_language", return_value="en_US"):
            translator = i18n.I18n()

        self.assertEqual(translator.language, "en_US")

    def test_missing_translation_falls_back_to_chinese_then_key(self):
        i18n.TRANSLATIONS["zh_CN"]["test.fallback"] = "回退文本"
        try:
            translator = i18n.I18n("en_US")
            self.assertEqual(translator("test.fallback"), "回退文本")
            self.assertEqual(translator("test.unknown"), "test.unknown")
        finally:
            del i18n.TRANSLATIONS["zh_CN"]["test.fallback"]

    def test_rejects_unsupported_manual_language(self):
        translator = i18n.I18n("zh_CN")
        with self.assertRaises(ValueError):
            translator.set_language("fr_FR")

    def test_process_restart_delay_is_bounded_exponential(self):
        self.assertEqual(
            [star_fire.calculate_restart_delay(attempt) for attempt in range(7)],
            [3, 6, 12, 24, 48, 60, 60],
        )

    def test_process_restart_schedule_respects_user_stop(self):
        root = mock.Mock()
        root.after.return_value = "timer-id"
        app = object.__new__(star_fire.StarFireAPP)
        app.root = root
        app.user_stopped = False
        app.restart_after_id = None
        app.restart_attempt = 0
        app.starfire_started_at = None
        app.starfire_running = False
        app.starfire_log = mock.Mock()

        app._schedule_starfire_restart()
        root.after.assert_called_once()
        self.assertEqual(root.after.call_args.args[0], 3000)
        self.assertEqual(app.restart_after_id, "timer-id")

        stopped = object.__new__(star_fire.StarFireAPP)
        stopped.root = mock.Mock()
        stopped.user_stopped = True
        stopped.restart_after_id = None
        stopped.restart_attempt = 0
        stopped.starfire_started_at = None
        stopped.starfire_running = False
        stopped.starfire_log = mock.Mock()
        stopped._schedule_starfire_restart()
        stopped.root.after.assert_not_called()

    def test_income_tcp_server_reports_connected_client_count(self):
        server = star_fire.IncomeTCPServer()
        self.assertEqual(server.get_client_count(), 0)

        server.clients.extend([mock.Mock(), mock.Mock()])
        self.assertEqual(server.get_client_count(), 2)

    def test_proxy_backend_message_contains_all_enabled_backends(self):
        class FakeTCPServer:
            def __init__(self):
                self.message = None

            def get_client_count(self):
                return 1

            def send_to_all_clients(self, message):
                self.message = message
                return 1

        app = object.__new__(star_fire.StarFireAPP)
        app.config = {'proxy_backends': [
            {'name': 'a', 'base_url': 'https://a.test/v1', 'api_key': 'key-a', 'enabled': True},
            {'name': 'b', 'base_url': 'https://b.test/v1', 'api_key': 'key-b', 'enabled': True},
            {'name': 'off', 'base_url': 'https://off.test/v1', 'api_key': 'key-off', 'enabled': False},
        ]}
        app.tcp_server = FakeTCPServer()
        app.pending_backend_message = None
        app.starfire_log = mock.Mock()

        self.assertEqual(app.send_proxy_backends_to_starfire(), 1)
        payload = json.loads(app.tcp_server.message)
        self.assertEqual(payload['type'], 'proxy_backends')
        self.assertEqual([item['name'] for item in payload['data']], ['a', 'b'])
        self.assertEqual(payload['data'][1]['api_key'], 'key-b')

    def test_proxy_backend_message_clears_backends_when_all_are_disabled(self):
        tcp_server = mock.Mock()
        tcp_server.get_client_count.return_value = 1
        tcp_server.send_to_all_clients.return_value = 1
        app = object.__new__(star_fire.StarFireAPP)
        app.config = {'proxy_backends': [
            {'name': 'off', 'base_url': 'https://off.test/v1', 'api_key': 'key', 'enabled': False},
        ]}
        app.tcp_server = tcp_server
        app.pending_backend_message = None
        app.starfire_log = mock.Mock()

        self.assertEqual(app.send_proxy_backends_to_starfire(), 1)
        payload = json.loads(tcp_server.send_to_all_clients.call_args.args[0])
        self.assertEqual(payload['data'], [])

    def test_source_translation_switches_back_to_original_chinese(self):
        translator = i18n.I18n("en_US")
        self.assertEqual(translator.translate_source("🔄 刷新"), "🔄 Refresh")
        translator.set_language("zh_CN")
        self.assertEqual(translator.translate_source("🔄 刷新"), "🔄 刷新")

    def test_selects_and_formats_runtime_template(self):
        translator = i18n.I18n("en_US")
        self.assertEqual(
            translator.select("已保存 {count} 个", "Saved {count}", count=3),
            "Saved 3",
        )

    def test_widget_refresh_preserves_canonical_source_text(self):
        class FakeWidget:
            def __init__(self, text, children=()):
                self.text = text
                self.children = children

            def cget(self, option):
                return self.text

            def configure(self, **options):
                self.text = options["text"]

            def winfo_children(self):
                return self.children

        child = FakeWidget("🔄 刷新")
        root = FakeWidget("", [child])
        translator = i18n.I18n("en_US")

        i18n.refresh_widget_texts(root, translator)
        self.assertEqual(child.text, "🔄 Refresh")
        translator.set_language("zh_CN")
        i18n.refresh_widget_texts(root, translator)
        self.assertEqual(child.text, "🔄 刷新")

    def test_widget_refresh_does_not_restore_stale_dynamic_text(self):
        class FakeWidget:
            def __init__(self):
                self.text = " ○ 未登录 "

            def cget(self, option):
                return self.text

            def configure(self, **options):
                self.text = options["text"]

            def winfo_children(self):
                return ()

        widget = FakeWidget()
        translator = i18n.I18n("zh_CN")
        i18n.refresh_widget_texts(widget, translator)
        widget.text = "125.50 ¥"
        translator.set_language("en_US")
        i18n.refresh_widget_texts(widget, translator)
        self.assertEqual(widget.text, "125.50 ¥")


if __name__ == "__main__":
    unittest.main()