#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Ollama 模型管理器
最小依赖的PC客户端程序，用于管理和运行 Ollama 模型
"""

import tkinter as tk
from tkinter import ttk, messagebox, scrolledtext
import subprocess
import platform
import webbrowser
import threading
import os
import sys
import re
import json
from datetime import datetime
import locale
import socket
import struct

# 导入音效模块
try:
    import winsound
    SOUND_AVAILABLE = True
except ImportError:
    SOUND_AVAILABLE = False

def get_resource_path(relative_path):
    """获取资源文件的绝对路径（支持打包后）"""
    try:
        # PyInstaller 创建临时文件夹，路径存储在 _MEIPASS
        base_path = sys._MEIPASS
    except Exception:
        # 开发环境下使用当前目录
        base_path = os.path.abspath(".")
    
    return os.path.join(base_path, relative_path)

# Windows 下隐藏控制台窗口的参数
if platform.system() == "Windows":
    SUBPROCESS_FLAGS = subprocess.CREATE_NO_WINDOW
else:
    SUBPROCESS_FLAGS = 0


def play_money_sound():
    """播放收钱音效"""
    if not SOUND_AVAILABLE:
        return
    
    try:
        # 在后台线程播放音效，避免阻塞UI
        def _play():
            try:
                # 使用系统默认的"叮"声
                # 可以替换为自定义wav文件: winsound.PlaySound("money.wav", winsound.SND_FILENAME | winsound.SND_ASYNC)
                winsound.MessageBeep(winsound.MB_ICONASTERISK)
            except:
                pass
        
        threading.Thread(target=_play, daemon=True).start()
    except:
        pass


# ============ Toast 通知类 ============
class ToastNotification:
    """优雅的Toast通知,用于显示收益等消息"""
    active_toasts = []  # 存储当前活动的toast
    
    def __init__(self, parent, message, title="通知", duration=4000, toast_type="info"):
        self.parent = parent
        self.duration = duration
        
        # 播放收钱音效(仅针对收益类型)
        if toast_type == "money":
            play_money_sound()
        
        # 创建顶层窗口
        self.toast = tk.Toplevel(parent)
        self.toast.overrideredirect(True)  # 无边框
        self.toast.attributes('-topmost', True)  # 置顶
        
        # 设置透明度(Windows)
        try:
            self.toast.attributes('-alpha', 0.95)
        except:
            pass
        
        # 配色方案
        colors = {
            'info': {'bg': '#3b82f6', 'fg': 'white'},
            'success': {'bg': '#10b981', 'fg': 'white'},
            'warning': {'bg': '#f59e0b', 'fg': 'white'},
            'error': {'bg': '#ef4444', 'fg': 'white'},
            'money': {'bg': '#10b981', 'fg': 'white'}  # 收益专用
        }
        
        color = colors.get(toast_type, colors['info'])
        
        # 主容器
        container = tk.Frame(self.toast, bg=color['bg'], padx=20, pady=15)
        container.pack(fill=tk.BOTH, expand=True)
        
        # 标题
        title_label = tk.Label(
            container,
            text=title,
            font=('Microsoft YaHei UI', 10, 'bold'),
            bg=color['bg'],
            fg=color['fg']
        )
        title_label.pack(anchor=tk.W)
        
        # 消息内容
        msg_label = tk.Label(
            container,
            text=message,
            font=('Microsoft YaHei UI', 9),
            bg=color['bg'],
            fg=color['fg'],
            wraplength=300,
            justify=tk.LEFT
        )
        msg_label.pack(anchor=tk.W, pady=(5, 0))
        
        # 更新窗口以获取实际大小
        self.toast.update_idletasks()
        
        # 计算位置(右下角)
        self._position_toast()
        
        # 绑定点击关闭
        container.bind('<Button-1>', lambda e: self.close())
        title_label.bind('<Button-1>', lambda e: self.close())
        msg_label.bind('<Button-1>', lambda e: self.close())
        
        # 添加到活动列表
        ToastNotification.active_toasts.append(self)
        
        # 滑入动画
        self._slide_in()
        
        # 自动关闭
        if duration > 0:
            self.toast.after(duration, self.close)
    
    def _position_toast(self):
        """定位toast到右下角,考虑已有toast的位置"""
        screen_width = self.parent.winfo_screenwidth()
        screen_height = self.parent.winfo_screenheight()
        
        toast_width = self.toast.winfo_width()
        toast_height = self.toast.winfo_height()
        
        # 右下角位置
        x = screen_width - toast_width - 20
        
        # 计算y位置,堆叠在其他toast上方
        y_offset = 20
        for toast in ToastNotification.active_toasts:
            if toast != self and toast.toast.winfo_exists():
                y_offset += toast.toast.winfo_height() + 10
        
        y = screen_height - toast_height - y_offset
        
        # 初始位置(屏幕外)
        self.start_x = screen_width
        self.end_x = x
        self.y = y
        
        self.toast.geometry(f'+{self.start_x}+{self.y}')
    
    def _slide_in(self):
        """滑入动画"""
        current_x = int(self.toast.winfo_x())
        if current_x > self.end_x:
            step = max(10, (current_x - self.end_x) // 10)
            new_x = current_x - step
            self.toast.geometry(f'+{new_x}+{self.y}')
            self.toast.after(10, self._slide_in)
        else:
            self.toast.geometry(f'+{self.end_x}+{self.y}')
    
    def _slide_out(self, callback):
        """滑出动画"""
        current_x = int(self.toast.winfo_x())
        screen_width = self.parent.winfo_screenwidth()
        if current_x < screen_width:
            step = max(10, (screen_width - current_x) // 10)
            new_x = current_x + step
            self.toast.geometry(f'+{new_x}+{self.y}')
            self.toast.after(10, lambda: self._slide_out(callback))
        else:
            callback()
    
    def close(self):
        """关闭toast"""
        if self in ToastNotification.active_toasts:
            ToastNotification.active_toasts.remove(self)
        
        def destroy():
            if self.toast.winfo_exists():
                self.toast.destroy()
        
        self._slide_out(destroy)


# ============ 收益消息解析函数 ============
def parse_income_message(line):
    """解析starfire输出中的收益消息
    返回: (is_income, amount, currency) 或 (False, None, None)
    """
    import re
    
    # 常见收益消息模式
    patterns = [
        r'收益[:\s]*([\d.]+)\s*([¥$元])',
        r'获得[:\s]*([\d.]+)\s*([¥$元])',
        r'赚取[:\s]*([\d.]+)\s*([¥$元])',
        r'income[:\s]*([\d.]+)\s*(CNY|USD|¥|\$)',
        r'earned[:\s]*([\d.]+)\s*(CNY|USD|¥|\$)',
        r'profit[:\s]*([\d.]+)\s*(CNY|USD|¥|\$)',
    ]
    
    line_lower = line.lower()
    for pattern in patterns:
        match = re.search(pattern, line, re.IGNORECASE)
        if match:
            amount = match.group(1)
            currency = match.group(2)
            return True, amount, currency
    
    return False, None, None


# ============ TCP服务器类 ============
class IncomeTCPServer:
    """TCP服务器,接收starfire.exe发送的收益消息"""
    
    def __init__(self, host='127.0.0.1', port=19527, callback=None):
        self.host = host
        self.port = port
        self.callback = callback  # 收到消息时的回调函数
        self.server_socket = None
        self.running = False
        self.server_thread = None
        
    def start(self):
        """启动TCP服务器"""
        if self.running:
            return False, "服务器已在运行中"
        
        try:
            self.server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            self.server_socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            self.server_socket.bind((self.host, self.port))
            self.server_socket.listen(5)
            self.running = True
            
            # 在后台线程中运行服务器
            self.server_thread = threading.Thread(target=self._run_server, daemon=True)
            self.server_thread.start()
            
            return True, f"TCP服务器已启动: {self.host}:{self.port}"
        except Exception as e:
            return False, f"启动失败: {str(e)}"
    
    def _run_server(self):
        """服务器主循环"""
        while self.running:
            try:
                # 设置超时,避免阻塞
                self.server_socket.settimeout(1.0)
                try:
                    client_socket, client_address = self.server_socket.accept()
                    # 在新线程中处理客户端连接
                    threading.Thread(
                        target=self._handle_client,
                        args=(client_socket, client_address),
                        daemon=True
                    ).start()
                except socket.timeout:
                    continue
            except Exception as e:
                if self.running:
                    if self.callback:
                        self.callback('error', f"服务器错误: {str(e)}")
                break
    
    def _handle_client(self, client_socket, client_address):
        """处理客户端连接"""
        try:
            if self.callback:
                self.callback('connect', f"客户端连接: {client_address}")
            
            while self.running:
                # 接收数据长度(4字节)
                length_data = client_socket.recv(4)
                if not length_data:
                    break
                
                # 解析长度
                message_length = struct.unpack('!I', length_data)[0]
                
                # 接收完整消息
                message_data = b''
                while len(message_data) < message_length:
                    chunk = client_socket.recv(message_length - len(message_data))
                    if not chunk:
                        break
                    message_data += chunk
                
                if len(message_data) == message_length:
                    # 解码消息
                    try:
                        message = message_data.decode('utf-8')
                        # 回调处理消息
                        if self.callback:
                            self.callback('message', message)
                    except Exception as e:
                        if self.callback:
                            self.callback('error', f"解码消息失败: {str(e)}")
        
        except Exception as e:
            if self.callback:
                self.callback('error', f"处理客户端错误: {str(e)}")
        finally:
            client_socket.close()
            if self.callback:
                self.callback('disconnect', f"客户端断开: {client_address}")
    
    def stop(self):
        """停止TCP服务器"""
        self.running = False
        if self.server_socket:
            try:
                self.server_socket.close()
            except:
                pass
        return True, "TCP服务器已停止"


# ============ 添加启动画面 ============
class SplashScreen:
    """启动画面，在主程序加载时显示"""
    def __init__(self):
        self.root = tk.Tk()
        self.root.overrideredirect(True)
        
        # 设置启动画面图标
        try:
            icon_path = get_resource_path("icon.ico")
            if os.path.exists(icon_path):
                self.root.iconbitmap(icon_path)
        except:
            pass
        
        width = 400
        height = 300
        screen_width = self.root.winfo_screenwidth()
        screen_height = self.root.winfo_screenheight()
        x = (screen_width - width) // 2
        y = (screen_height - height) // 2
        self.root.geometry(f'{width}x{height}+{x}+{y}')
        
        self.root.configure(bg='#2C3E50')
        
        main_frame = tk.Frame(self.root, bg='#2C3E50')
        main_frame.pack(expand=True, fill='both', padx=20, pady=20)
        
        title_label = tk.Label(
            main_frame,
            text="StarFire MaaS",
            font=('Arial', 24, 'bold'),
            bg='#2C3E50',
            fg='#ECF0F1'
        )
        title_label.pack(pady=(20, 10))
        
        subtitle_label = tk.Label(
            main_frame,
            text="算力分享应用",
            font=('Arial', 12),
            bg='#2C3E50',
            fg='#BDC3C7'
        )
        subtitle_label.pack(pady=(0, 30))
        
        self.progress = ttk.Progressbar(
            main_frame,
            mode='indeterminate',
            length=300
        )
        self.progress.pack(pady=20)
        self.progress.start(10)
        
        self.status_label = tk.Label(
            main_frame,
            text="正在启动...",
            font=('Arial', 10),
            bg='#2C3E50',
            fg='#95A5A6'
        )
        self.status_label.pack(pady=10)
        
        version_label = tk.Label(
            main_frame,
            text="v1.0.0",
            font=('Arial', 8),
            bg='#2C3E50',
            fg='#7F8C8D'
        )
        version_label.pack(side='bottom', pady=10)
        
        self.root.update()
    
    def update_status(self, text):
        self.status_label.config(text=text)
        self.root.update()
    
    def close(self):
        self.progress.stop()
        self.root.destroy()


class StarFireAPP:
    def __init__(self, root):
        self.root = root
        self.root.title("StarFire MaaS 算力分享APP")
        self.root.geometry("1000x700")
        self.root.resizable(True, True)
        
        # 设置窗口关闭事件
        self.root.protocol("WM_DELETE_WINDOW", self.on_closing)
        
        # 设置窗口图标
        try:
            icon_path = get_resource_path("icon.ico")
            if os.path.exists(icon_path):
                self.root.iconbitmap(icon_path)
        except Exception as e:
            print(f"设置图标失败: {e}")
        
        self.running_process = None
        self.selected_model = None
        self.model_thread = None
        self.running_models = set()
        self.starfire_process = None
        self.starfire_running = False
        self.total_income = 0.0  # 累计收益
        
        # 创建TCP服务器并自动启动
        self.tcp_server = IncomeTCPServer(
            host='127.0.0.1',
            port=19527,
            callback=self.handle_tcp_message
        )
        # 自动启动TCP服务器
        success, msg = self.tcp_server.start()
        if not success:
            print(f"TCP服务器启动失败: {msg}")
        
        self.model_categories = {
            'embedding': ['embed', 'nomic-embed', 'mxbai-embed', 'bge-', 'gte-'],
            'reranker': ['rerank', 'bge-reranker'],
            'vision': ['llava', 'bakllava', 'vision', 'moondream', 'clip'],
            'code': ['codellama', 'starcoder', 'codegemma', 'deepseek-coder', 'qwen-coder'],
            'chat': []
        }
        
        self.config_file = "starfire_config.json"
        self.load_config()
        
        self.create_widgets()
        self.check_ollama()
        self.check_running_models()
    
    def load_config(self):
        self.config = {
            'host': '115.190.26.60',
            'token': '',
            'ippm': '3.8',
            'oppm': '8.3',
            'model_mode': 'ollama',  # ollama, vllm, proxy, llamacpp
            'proxy_base_url': 'http://localhost:8000/v1',
            'proxy_api_key': '',
            'ollama_num_parallel': ''  # Ollama并发请求数
        }
        
        try:
            if os.path.exists(self.config_file):
                with open(self.config_file, 'r', encoding='utf-8') as f:
                    saved_config = json.load(f)
                    self.config.update(saved_config)
        except:
            pass
        
        # 备份原始配置,用于检测修改
        self.original_config = self.config.copy()
    
    def save_config(self):
        try:
            # 仅在手动保存时备份配置到历史目录
            # 判断是否是自动保存(通过检查调用栈)
            import traceback
            stack = traceback.extract_stack()
            is_auto_save = any('auto_save_config' in frame.name for frame in stack)
            
            if not is_auto_save:
                # 手动保存时才备份到历史目录
                history_dir = "config_history"
                if not os.path.exists(history_dir):
                    os.makedirs(history_dir)
                
                timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
                history_file = os.path.join(history_dir, f"starfire_config_{timestamp}.json")
                
                # 如果配置文件已存在,先备份
                if os.path.exists(self.config_file):
                    try:
                        with open(self.config_file, 'r', encoding='utf-8') as f:
                            old_config = json.load(f)
                        with open(history_file, 'w', encoding='utf-8') as f:
                            json.dump(old_config, f, indent=2, ensure_ascii=False)
                    except:
                        pass
            
            # 保存新配置
            with open(self.config_file, 'w', encoding='utf-8') as f:
                json.dump(self.config, f, indent=2, ensure_ascii=False)
            
            # 更新原始配置备份
            self.original_config = self.config.copy()
        except Exception as e:
            self.log(f"保存配置失败: {str(e)}", "red")
    
    def auto_save_config(self, field_name):
        """自动检测配置修改并保存"""
        # 获取当前输入框的值
        current_values = {
            'host': self.host_entry.get().strip(),
            'token': self.token_entry.get().strip(),
            'ippm': self.ippm_entry.get().strip(),
            'oppm': self.oppm_entry.get().strip(),
            'proxy_base_url': self.proxy_base_url_entry.get().strip(),
            'proxy_api_key': self.proxy_api_key_entry.get().strip(),
            'ollama_num_parallel': self.ollama_num_parallel_entry.get().strip()
        }
        
        # 检查是否有修改
        if field_name in current_values and current_values[field_name] != self.original_config.get(field_name, ''):
            # 更新配置
            self.config[field_name] = current_values[field_name]
            # 保存到文件
            self.save_config()
            self.starfire_log(f"✓ 配置已自动保存: {field_name}", "green")
    
    def get_model_category(self, model_name):
        model_lower = model_name.lower()
        for category, keywords in self.model_categories.items():
            for keyword in keywords:
                if keyword in model_lower:
                    return category
        return 'chat'
    
    def get_category_icon(self, category):
        icons = {
            'embedding': '📊',
            'reranker': '🔍',
            'vision': '👁️',
            'code': '💻',
            'chat': '💬'
        }
        return icons.get(category, '💬')
    
    def get_category_name(self, category):
        names = {
            'embedding': 'Embedding',
            'reranker': 'Reranker',
            'vision': '多模态',
            'code': '代码',
            'chat': '对话'
        }
        return names.get(category, '对话')
    
    def on_mode_change(self):
        """模型接入方式变更时的回调"""
        mode = self.model_mode_var.get()
        
        # 自动保存模型模式变更
        if mode != self.original_config.get('model_mode', 'ollama'):
            self.config['model_mode'] = mode
            self.save_config()
            self.starfire_log(f"✓ 模型接入方式已自动保存: {mode}", "green")
        
        # 显示/隐藏配置
        if mode == 'proxy':
            self.proxy_config_frame.pack(fill=tk.X, pady=(10, 0))
            self.ollama_config_frame.pack_forget()
            self.status_label.config(
                text="✓ 代理模式 - 请配置 Base URL 和 API Key",
                foreground="blue"
            )
            # 在代理模式下禁用Ollama相关按钮
            self.refresh_btn.config(state=tk.DISABLED)
            self.run_btn.config(state=tk.DISABLED)
            self.stop_btn.config(state=tk.DISABLED)
            self.log("已切换到代理模式", "blue")
        elif mode == 'ollama':
            self.proxy_config_frame.pack_forget()
            self.ollama_config_frame.pack(fill=tk.X, pady=(10, 0))
            self.refresh_btn.config(state=tk.NORMAL)
            self.check_ollama()
            self.log("已切换到 Ollama 模式", "blue")
        elif mode == 'vllm':
            self.proxy_config_frame.pack_forget()
            self.ollama_config_frame.pack_forget()
            self.status_label.config(
                text="vLLM 模式开发中...",
                foreground="orange"
            )
            self.refresh_btn.config(state=tk.DISABLED)
            self.run_btn.config(state=tk.DISABLED)
            self.stop_btn.config(state=tk.DISABLED)
        elif mode == 'llamacpp':
            self.proxy_config_frame.pack_forget()
            self.ollama_config_frame.pack_forget()
            self.status_label.config(
                text="llama.cpp 模式开发中...",
                foreground="orange"
            )
            self.refresh_btn.config(state=tk.DISABLED)
            self.run_btn.config(state=tk.DISABLED)
            self.stop_btn.config(state=tk.DISABLED)
    
    def create_widgets(self):
        main_paned = ttk.PanedWindow(self.root, orient=tk.HORIZONTAL)
        main_paned.pack(fill=tk.BOTH, expand=True, padx=5, pady=5)
        
        left_frame = ttk.Frame(main_paned)
        main_paned.add(left_frame, weight=6)
        
        right_frame = ttk.Frame(main_paned)
        main_paned.add(right_frame, weight=4)
        
        # 左侧
        top_frame = ttk.Frame(left_frame, padding="10")
        top_frame.pack(fill=tk.X)
        
        # 模型接入方式选择
        mode_frame = ttk.LabelFrame(top_frame, text="🔌 模型接入方式", padding="10")
        mode_frame.pack(fill=tk.X, pady=(0, 10))
        
        self.model_mode_var = tk.StringVar(value=self.config.get('model_mode', 'ollama'))
        
        modes_container = ttk.Frame(mode_frame)
        modes_container.pack(fill=tk.X)
        
        ttk.Radiobutton(
            modes_container,
            text="Ollama (本地)",
            variable=self.model_mode_var,
            value="ollama",
            command=self.on_mode_change
        ).pack(side=tk.LEFT, padx=10)
        
        ttk.Radiobutton(
            modes_container,
            text="vLLM (开发中)",
            variable=self.model_mode_var,
            value="vllm",
            command=self.on_mode_change,
            state=tk.DISABLED
        ).pack(side=tk.LEFT, padx=10)
        
        ttk.Radiobutton(
            modes_container,
            text="llama.cpp (开发中)",
            variable=self.model_mode_var,
            value="llamacpp",
            command=self.on_mode_change,
            state=tk.DISABLED
        ).pack(side=tk.LEFT, padx=10)
        
        ttk.Radiobutton(
            modes_container,
            text="代理模式",
            variable=self.model_mode_var,
            value="proxy",
            command=self.on_mode_change
        ).pack(side=tk.LEFT, padx=10)
        
        # 代理模式配置（初始隐藏）
        self.proxy_config_frame = ttk.Frame(mode_frame)
        
        proxy_url_frame = ttk.Frame(self.proxy_config_frame)
        proxy_url_frame.pack(fill=tk.X, pady=5)
        ttk.Label(proxy_url_frame, text="Base URL:", width=10).pack(side=tk.LEFT)
        self.proxy_base_url_entry = ttk.Entry(proxy_url_frame)
        self.proxy_base_url_entry.insert(0, self.config.get('proxy_base_url', 'http://localhost:8000/v1'))
        self.proxy_base_url_entry.pack(side=tk.LEFT, fill=tk.X, expand=True, padx=(5, 0))
        self.proxy_base_url_entry.bind('<FocusOut>', lambda e: self.auto_save_config('proxy_base_url'))
        
        proxy_key_frame = ttk.Frame(self.proxy_config_frame)
        proxy_key_frame.pack(fill=tk.X, pady=5)
        ttk.Label(proxy_key_frame, text="API Key:", width=10).pack(side=tk.LEFT)
        self.proxy_api_key_entry = ttk.Entry(proxy_key_frame, show="*")
        self.proxy_api_key_entry.insert(0, self.config.get('proxy_api_key', ''))
        self.proxy_api_key_entry.pack(side=tk.LEFT, fill=tk.X, expand=True, padx=(5, 0))
        self.proxy_api_key_entry.bind('<FocusOut>', lambda e: self.auto_save_config('proxy_api_key'))
        
        def toggle_proxy_key():
            if self.proxy_api_key_entry['show'] == '*':
                self.proxy_api_key_entry['show'] = ''
                toggle_proxy_btn.config(text="👁️")
            else:
                self.proxy_api_key_entry['show'] = '*'
                toggle_proxy_btn.config(text="🔒")
        
        toggle_proxy_btn = ttk.Button(proxy_key_frame, text="🔒", width=3, command=toggle_proxy_key)
        toggle_proxy_btn.pack(side=tk.LEFT, padx=(5, 0))
        
        # 根据当前模式显示/隐藏代理配置
        if self.model_mode_var.get() == 'proxy':
            self.proxy_config_frame.pack(fill=tk.X, pady=(10, 0))
        
        # Ollama 并发设置（仅在 Ollama 模式下显示）
        self.ollama_config_frame = ttk.Frame(mode_frame)
        
        ollama_parallel_frame = ttk.Frame(self.ollama_config_frame)
        ollama_parallel_frame.pack(fill=tk.X, pady=5)
        ttk.Label(ollama_parallel_frame, text="并发请求数:", width=10).pack(side=tk.LEFT)
        self.ollama_num_parallel_entry = ttk.Entry(ollama_parallel_frame, width=10)
        self.ollama_num_parallel_entry.insert(0, self.config.get('ollama_num_parallel', ''))
        self.ollama_num_parallel_entry.pack(side=tk.LEFT, padx=(5, 5))
        self.ollama_num_parallel_entry.bind('<FocusOut>', lambda e: self.auto_save_config('ollama_num_parallel'))
        
        ttk.Label(
            ollama_parallel_frame, 
            text="(空值=自动，推荐4或1)",
            foreground="gray",
            font=("Arial", 8)
        ).pack(side=tk.LEFT)
        
        # 提示信息
        ollama_tip = ttk.Label(
            self.ollama_config_frame,
            text="💡 每个模型同时处理的最大并行请求数。默认根据可用内存自动选择4或1",
            foreground="#666",
            font=("Arial", 8),
            wraplength=400
        )
        ollama_tip.pack(fill=tk.X, pady=(0, 5))
        
        # 根据当前模式显示/隐藏Ollama配置
        if self.model_mode_var.get() == 'ollama':
            self.ollama_config_frame.pack(fill=tk.X, pady=(10, 0))
        
        self.status_label = ttk.Label(
            top_frame, 
            text="正在检查模型服务状态...", 
            font=("Arial", 10)
        )
        self.status_label.pack(anchor=tk.W, pady=(10, 0))
        
        list_frame = ttk.LabelFrame(left_frame, text="📦 已安装的模型", padding="10")
        list_frame.pack(fill=tk.BOTH, expand=True, padx=10, pady=5)
        
        tree_container = ttk.Frame(list_frame)
        tree_container.pack(fill=tk.BOTH, expand=True)
        
        columns = ("分类", "模型名称", "大小", "修改时间")
        self.model_tree = ttk.Treeview(
            tree_container, 
            columns=columns, 
            show="headings", 
            height=12
        )
        
        self.model_tree.heading("分类", text="分类")
        self.model_tree.heading("模型名称", text="模型名称")
        self.model_tree.heading("大小", text="大小")
        self.model_tree.heading("修改时间", text="修改时间")
        
        self.model_tree.column("分类", width=100, anchor=tk.CENTER)
        self.model_tree.column("模型名称", width=180)
        self.model_tree.column("大小", width=80, anchor=tk.CENTER)
        self.model_tree.column("修改时间", width=150, anchor=tk.CENTER)
        
        scrollbar = ttk.Scrollbar(tree_container, orient=tk.VERTICAL, command=self.model_tree.yview)
        self.model_tree.configure(yscrollcommand=scrollbar.set)
        
        self.model_tree.pack(side=tk.LEFT, fill=tk.BOTH, expand=True)
        scrollbar.pack(side=tk.RIGHT, fill=tk.Y)
        
        legend_frame = ttk.Frame(list_frame)
        legend_frame.pack(fill=tk.X, pady=(5, 0))
        
        ttk.Label(legend_frame, text="状态:", font=("Arial", 9, "bold")).pack(side=tk.LEFT, padx=(0, 5))
        
        running_legend = tk.Label(
            legend_frame, 
            text=" ● 运行中 ", 
            bg="#90EE90", 
            fg="darkgreen",
            relief=tk.RAISED,
            padx=5
        )
        running_legend.pack(side=tk.LEFT, padx=5)
        
        idle_legend = tk.Label(
            legend_frame, 
            text=" ○ 未运行 ", 
            bg="#D3D3D3", 
            fg="gray",
            relief=tk.RAISED,
            padx=5
        )
        idle_legend.pack(side=tk.LEFT, padx=5)
        
        self.running_label = ttk.Label(
            legend_frame,
            text="",
            foreground="green",
            font=("Arial", 9, "bold")
        )
        self.running_label.pack(side=tk.LEFT, padx=10)
        
        button_frame = ttk.Frame(left_frame, padding="10")
        button_frame.pack(fill=tk.X)
        
        self.refresh_btn = ttk.Button(
            button_frame, 
            text="🔄 刷新", 
            command=self.load_models,
            width=12
        )
        self.refresh_btn.pack(side=tk.LEFT, padx=5)
        
        self.run_btn = ttk.Button(
            button_frame, 
            text="▶️ 运行", 
            command=self.run_model,
            state=tk.DISABLED,
            width=12
        )
        self.run_btn.pack(side=tk.LEFT, padx=5)
        
        self.stop_btn = ttk.Button(
            button_frame, 
            text="⏹️ 停止", 
            command=self.stop_model,
            state=tk.DISABLED,
            width=12
        )
        self.stop_btn.pack(side=tk.LEFT, padx=5)
        
        # 测试Toast通知按钮
        test_toast_btn = ttk.Button(
            button_frame,
            text="🔔 测试通知",
            command=self.test_toast_notification,
            width=12
        )
        test_toast_btn.pack(side=tk.LEFT, padx=5)
        
        log_frame = ttk.LabelFrame(left_frame, text="📋 运行日志", padding="10")
        log_frame.pack(fill=tk.BOTH, expand=True, padx=10, pady=5)
        
        self.log_text = scrolledtext.ScrolledText(
            log_frame, 
            height=8, 
            state=tk.DISABLED, 
            wrap=tk.WORD,
            font=("Consolas", 9)
        )
        self.log_text.pack(fill=tk.BOTH, expand=True)
        
        # 右侧
        starfire_title = ttk.Frame(right_frame, padding="10")
        starfire_title.pack(fill=tk.X)
        
        ttk.Label(
            starfire_title,
            text="🌟 Starfire 算力注册",
            font=("Arial", 12, "bold")
        ).pack(anchor=tk.W)
        
        config_frame = ttk.LabelFrame(right_frame, text="⚙️ 配置参数", padding="15")
        config_frame.pack(fill=tk.X, padx=10, pady=5)
        
        host_frame = ttk.Frame(config_frame)
        host_frame.pack(fill=tk.X, pady=5)
        ttk.Label(host_frame, text="服务器地址:", width=12).pack(side=tk.LEFT)
        self.host_entry = ttk.Entry(host_frame)
        self.host_entry.insert(0, self.config['host'])
        self.host_entry.pack(side=tk.LEFT, fill=tk.X, expand=True, padx=(5, 0))
        self.host_entry.bind('<FocusOut>', lambda e: self.auto_save_config('host'))
        
        token_frame = ttk.Frame(config_frame)
        token_frame.pack(fill=tk.X, pady=5)
        ttk.Label(token_frame, text="Token:", width=12).pack(side=tk.LEFT)
        self.token_entry = ttk.Entry(token_frame, show="*")
        self.token_entry.insert(0, self.config['token'])
        self.token_entry.pack(side=tk.LEFT, fill=tk.X, expand=True, padx=(5, 0))
        self.token_entry.bind('<FocusOut>', lambda e: self.auto_save_config('token'))
        
        def toggle_token():
            if self.token_entry['show'] == '*':
                self.token_entry['show'] = ''
                toggle_btn.config(text="👁️")
            else:
                self.token_entry['show'] = '*'
                toggle_btn.config(text="🔒")
        
        toggle_btn = ttk.Button(token_frame, text="🔒", width=3, command=toggle_token)
        toggle_btn.pack(side=tk.LEFT, padx=(5, 0))
        
        ippm_frame = ttk.Frame(config_frame)
        ippm_frame.pack(fill=tk.X, pady=5)
        ttk.Label(ippm_frame, text="输入价格:", width=12).pack(side=tk.LEFT)
        self.ippm_entry = ttk.Entry(ippm_frame, width=15)
        self.ippm_entry.insert(0, self.config['ippm'])
        self.ippm_entry.pack(side=tk.LEFT, padx=(5, 0))
        self.ippm_entry.bind('<FocusOut>', lambda e: self.auto_save_config('ippm'))
        ttk.Label(ippm_frame, text="¥/M tokens").pack(side=tk.LEFT, padx=(5, 0))
        
        oppm_frame = ttk.Frame(config_frame)
        oppm_frame.pack(fill=tk.X, pady=5)
        ttk.Label(oppm_frame, text="输出价格:", width=12).pack(side=tk.LEFT)
        self.oppm_entry = ttk.Entry(oppm_frame, width=15)
        self.oppm_entry.insert(0, self.config['oppm'])
        self.oppm_entry.pack(side=tk.LEFT, padx=(5, 0))
        self.oppm_entry.bind('<FocusOut>', lambda e: self.auto_save_config('oppm'))
        ttk.Label(oppm_frame, text="¥/M tokens").pack(side=tk.LEFT, padx=(5, 0))
        
        starfire_button_frame = ttk.Frame(config_frame)
        starfire_button_frame.pack(fill=tk.X, pady=(10, 0))
        
        self.save_config_btn = ttk.Button(
            starfire_button_frame,
            text="💾 保存配置",
            command=self.save_config_action,
            width=15
        )
        self.save_config_btn.pack(side=tk.LEFT, padx=5)
        
        self.register_btn = ttk.Button(
            starfire_button_frame,
            text="🚀 获取Token",
            command=self.open_starfire,
            width=15
        )
        self.register_btn.pack(side=tk.LEFT, padx=5)
        
        control_frame = ttk.LabelFrame(right_frame, text="🎮 算力控制", padding="15")
        control_frame.pack(fill=tk.X, padx=10, pady=5)
        
        status_indicator_frame = ttk.Frame(control_frame)
        status_indicator_frame.pack(fill=tk.X, pady=(0, 10))
        
        ttk.Label(status_indicator_frame, text="状态:", font=("Arial", 10, "bold")).pack(side=tk.LEFT)
        self.starfire_status_label = tk.Label(
            status_indicator_frame,
            text=" ● 未运行 ",
            bg="#D3D3D3",
            fg="gray",
            relief=tk.RAISED,
            padx=10,
            font=("Arial", 10, "bold")
        )
        self.starfire_status_label.pack(side=tk.LEFT, padx=10)
        
        # TCP服务器状态
        self.tcp_status_label = tk.Label(
            status_indicator_frame,
            text=" ○ TCP未启动 ",
            bg="#D3D3D3",
            fg="gray",
            relief=tk.RAISED,
            padx=8,
            font=("Arial", 9)
        )
        self.tcp_status_label.pack(side=tk.LEFT, padx=5)
        
        # 更新TCP状态为运行中
        self.tcp_status_label.config(
            text=" ● TCP运行中 ",
            bg="#90EE90",
            fg="green"
        )
        
        control_buttons = ttk.Frame(control_frame)
        control_buttons.pack(fill=tk.X)
        
        # TCP服务器信息
        tcp_info_label = ttk.Label(
            control_frame,
            text="💡 TCP服务器地址: 127.0.0.1:19527 (自动启动)",
            foreground="gray",
            font=("Arial", 8)
        )
        tcp_info_label.pack(pady=(0, 10))
        
        # Starfire控制按钮
        self.start_starfire_btn = ttk.Button(
            control_buttons,
            text="▶️ 启动算力注册",
            command=self.start_starfire,
            width=20
        )
        self.start_starfire_btn.pack(side=tk.LEFT, padx=5)
        
        self.stop_starfire_btn = ttk.Button(
            control_buttons,
            text="⏹️ 停止算力注册",
            command=self.stop_starfire,
            state=tk.DISABLED,
            width=20
        )
        self.stop_starfire_btn.pack(side=tk.LEFT, padx=5)
        
        starfire_log_frame = ttk.LabelFrame(right_frame, text="📊 Starfire 日志", padding="10")
        starfire_log_frame.pack(fill=tk.BOTH, expand=True, padx=10, pady=5)
        
        self.starfire_log_text = scrolledtext.ScrolledText(
            starfire_log_frame,
            height=15,
            state=tk.DISABLED,
            wrap=tk.WORD,
            font=("Consolas", 9)
        )
        self.starfire_log_text.pack(fill=tk.BOTH, expand=True)
        
        help_frame = ttk.Frame(right_frame, padding="10")
        help_frame.pack(fill=tk.X)
        
        help_text = "💡 提示: 需要 starfire.exe 与本程序在同一目录"
        ttk.Label(help_frame, text=help_text, foreground="gray", font=("Arial", 8)).pack()
    
    def log(self, message, color=None):
        self.log_text.config(state=tk.NORMAL)
        if color:
            tag = f"color_{color}"
            self.log_text.tag_config(tag, foreground=color)
            self.log_text.insert(tk.END, f"{message}\n", tag)
        else:
            self.log_text.insert(tk.END, f"{message}\n")
        self.log_text.see(tk.END)
        self.log_text.config(state=tk.DISABLED)
    
    def starfire_log(self, message, color=None):
        def _log():
            self.starfire_log_text.config(state=tk.NORMAL)
            timestamp = datetime.now().strftime("%H:%M:%S")
            
            if color:
                tag = f"sf_color_{color}"
                self.starfire_log_text.tag_config(tag, foreground=color)
                self.starfire_log_text.insert(tk.END, f"[{timestamp}] {message}\n", tag)
            else:
                self.starfire_log_text.insert(tk.END, f"[{timestamp}] {message}\n")
            
            self.starfire_log_text.see(tk.END)
            self.starfire_log_text.config(state=tk.DISABLED)
        
        if threading.current_thread() != threading.main_thread():
            self.root.after(0, _log)
        else:
            _log()
    
    def save_config_action(self):
        self.config['host'] = self.host_entry.get().strip()
        self.config['token'] = self.token_entry.get().strip()
        self.config['ippm'] = self.ippm_entry.get().strip()
        self.config['oppm'] = self.oppm_entry.get().strip()
        self.config['model_mode'] = self.model_mode_var.get()
        self.config['proxy_base_url'] = self.proxy_base_url_entry.get().strip()
        self.config['proxy_api_key'] = self.proxy_api_key_entry.get().strip()
        self.config['ollama_num_parallel'] = self.ollama_num_parallel_entry.get().strip()
        
        self.save_config()
        self.starfire_log("✓ 配置已保存", "green")
        messagebox.showinfo("成功", "配置已保存！")
    
    def on_closing(self):
        """窗口关闭时的清理工作"""
        # 停止TCP服务器
        if hasattr(self, 'tcp_server'):
            self.tcp_server.stop()
        
        # 停止Starfire进程
        if self.starfire_running and self.starfire_process:
            try:
                self.starfire_process.terminate()
                self.starfire_process.wait(timeout=3)
            except:
                pass
        
        # 关闭窗口
        self.root.destroy()
    
    def start_starfire(self):
        host = self.host_entry.get().strip()
        token = self.token_entry.get().strip()
        ippm = self.ippm_entry.get().strip()
        oppm = self.oppm_entry.get().strip()
        model_mode = self.model_mode_var.get()
        
        if not all([host, token, ippm, oppm]):
            messagebox.showwarning("配置不完整", "请填写所有必填配置项！")
            return
        
        # 代理模式需要额外检查配置
        if model_mode == 'proxy':
            proxy_url = self.proxy_base_url_entry.get().strip()
            proxy_key = self.proxy_api_key_entry.get().strip()
            if not all([proxy_url, proxy_key]):
                messagebox.showwarning("配置不完整", "代理模式需要配置 Base URL 和 API Key！")
                return
        
        #starfire_exe = "starfire.exe" if platform.system() == "Windows" else "./starfire"
        # 改为：
        if platform.system() == "Windows":
            starfire_exe = get_resource_path("starfire.exe")
        else:
            starfire_exe = get_resource_path("starfire")
        
        if not os.path.exists(starfire_exe):
            messagebox.showerror(
                "文件不存在",
                f"未找到 {starfire_exe}\n请将 starfire 可执行文件放在程序同一目录下"
            )
            return
        
        try:
            # 基础命令参数
            cmd = [
                starfire_exe,
                "-host", host,
                "-token", token,
                "-ippm", ippm,
                "-oppm", oppm
            ]
            
            # 根据模型模式添加额外参数
            if model_mode == 'proxy':
                proxy_url = self.proxy_base_url_entry.get().strip()
                proxy_key = self.proxy_api_key_entry.get().strip()
                cmd.extend([
                    "-engine", "openai",
                    "-openai-url", proxy_url,
                    "-openai-key", proxy_key
                ])
            
            self.starfire_log("=" * 50, "blue")
            self.starfire_log(f"正在启动 Starfire 算力注册...", "blue")
            self.starfire_log(f"模型模式: {model_mode}", "blue")
            self.starfire_log(f"服务器: {host}", "blue")
            if model_mode == 'proxy':
                self.starfire_log(f"代理地址: {proxy_url}", "blue")
            self.starfire_log(f"输入价格: {ippm} ¥/M tokens", "blue")
            self.starfire_log(f"输出价格: {oppm} ¥/M tokens", "blue")
            
            # 设置环境变量
            env = os.environ.copy()
            ollama_parallel = self.ollama_num_parallel_entry.get().strip()
            if ollama_parallel and model_mode == 'ollama':
                env['OLLAMA_NUM_PARALLEL'] = ollama_parallel
                self.starfire_log(f"并发请求数: {ollama_parallel}", "blue")
            
            self.starfire_log("=" * 50, "blue")
            
            if platform.system() == "Windows":
                self.starfire_process = subprocess.Popen(
                    cmd,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.STDOUT,
                    bufsize=0,
                    creationflags=SUBPROCESS_FLAGS,
                    env=env
                )
            else:
                self.starfire_process = subprocess.Popen(
                    cmd,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.STDOUT,
                    text=True,
                    bufsize=1,
                    universal_newlines=True,
                    env=env
                )
            
            self.starfire_running = True
            
            self.start_starfire_btn.config(state=tk.DISABLED)
            self.stop_starfire_btn.config(state=tk.NORMAL)
            self.starfire_status_label.config(
                text=" ● 运行中 ",
                bg="#90EE90",
                fg="darkgreen"
            )
            
            self.starfire_log(f"✓ Starfire 进程已启动", "green")
            self.starfire_log("开始接收日志输出...\n", "gray")
            
            threading.Thread(target=self._read_starfire_output, daemon=True).start()
            
        except Exception as e:
            self.starfire_log(f"✗ 启动失败: {str(e)}", "red")
            messagebox.showerror("启动失败", f"无法启动 Starfire:\n{str(e)}")
    
    def _read_starfire_output(self):
        try:
            if platform.system() == "Windows":
                while self.starfire_running and self.starfire_process:
                    line_bytes = b''
                    while self.starfire_running and self.starfire_process:
                        byte = self.starfire_process.stdout.read(1)
                        if not byte:
                            if self.starfire_process.poll() is not None:
                                break
                            continue
                        
                        if byte == b'\n':
                            break
                        line_bytes += byte
                    
                    if line_bytes:
                        line = None
                        for encoding in ['utf-8', 'gbk', 'gb2312', 'latin1']:
                            try:
                                line = line_bytes.decode(encoding).rstrip()
                                break
                            except:
                                continue
                        
                        if line is None:
                            line = line_bytes.decode('utf-8', errors='ignore').rstrip()
                        
                        if line:
                            # 检测收益消息
                            is_income, amount, currency = parse_income_message(line)
                            if is_income:
                                self.starfire_log(line, "green")
                                # 显示toast通知
                                self.total_income += float(amount)
                                self.show_income_toast(amount, currency)
                            elif any(keyword in line.lower() for keyword in ['error', 'failed', '失败', '错误']):
                                self.starfire_log(line, "red")
                            elif any(keyword in line.lower() for keyword in ['success', 'connected', '成功', '连接']):
                                self.starfire_log(line, "green")
                            elif any(keyword in line.lower() for keyword in ['warning', '警告']):
                                self.starfire_log(line, "orange")
                            elif any(keyword in line.lower() for keyword in ['info', '信息', 'request', '请求']):
                                self.starfire_log(line, "blue")
                            else:
                                self.starfire_log(line)
                    
                    if self.starfire_process.poll() is not None:
                        break
            else:
                while self.starfire_running and self.starfire_process:
                    line = self.starfire_process.stdout.readline()
                    
                    if line:
                        line = line.rstrip()
                        
                        # 检测收益消息
                        is_income, amount, currency = parse_income_message(line)
                        if is_income:
                            self.starfire_log(line, "green")
                            # 显示toast通知
                            self.total_income += float(amount)
                            self.show_income_toast(amount, currency)
                        elif any(keyword in line.lower() for keyword in ['error', 'failed', '失败', '错误']):
                            self.starfire_log(line, "red")
                        elif any(keyword in line.lower() for keyword in ['success', 'connected', '成功', '连接']):
                            self.starfire_log(line, "green")
                        elif any(keyword in line.lower() for keyword in ['warning', '警告']):
                            self.starfire_log(line, "orange")
                        elif any(keyword in line.lower() for keyword in ['info', '信息', 'request', '请求']):
                            self.starfire_log(line, "blue")
                        else:
                            self.starfire_log(line)
                    elif self.starfire_process.poll() is not None:
                        break
            
            if self.starfire_process:
                return_code = self.starfire_process.returncode
                self.starfire_log("\n" + "=" * 50, "gray")
                
                if return_code == 0:
                    self.starfire_log(f"✓ Starfire 已正常停止 (退出码: {return_code})", "green")
                else:
                    self.starfire_log(f"✗ Starfire 异常退出 (退出码: {return_code})", "red")
                
                self.starfire_log("=" * 50, "gray")
                
        except Exception as e:
            self.starfire_log(f"\n✗ 读取输出时出错: {str(e)}", "red")
        finally:
            self.root.after(0, self._reset_starfire_ui)
    
    def stop_starfire(self):
        if self.starfire_process:
            try:
                self.starfire_log("\n" + "=" * 50, "orange")
                self.starfire_log("正在停止 Starfire...", "orange")
                self.starfire_running = False
                
                self.starfire_process.terminate()
                
                try:
                    self.starfire_process.wait(timeout=5)
                    self.starfire_log("✓ Starfire 已正常停止", "green")
                except subprocess.TimeoutExpired:
                    self.starfire_log("强制终止 Starfire 进程...", "red")
                    self.starfire_process.kill()
                    self.starfire_process.wait()
                    self.starfire_log("✓ Starfire 已强制停止", "orange")
                
                self.starfire_log("=" * 50 + "\n", "orange")
                
                self.starfire_process = None
                self._reset_starfire_ui()
                
            except Exception as e:
                self.starfire_log(f"✗ 停止时出错: {str(e)}", "red")
    
    def handle_tcp_message(self, msg_type, content):
        """处理TCP服务器接收到的消息"""
        if msg_type == 'connect':
            self.starfire_log(f"🔗 {content}", "blue")
        elif msg_type == 'disconnect':
            self.starfire_log(f"🔌 {content}", "gray")
        elif msg_type == 'error':
            self.starfire_log(f"❌ {content}", "red")
        elif msg_type == 'message':
            # 输出原始消息用于调试
            self.starfire_log(f"🔍 [DEBUG] 收到原始消息: {content}", "purple")
            
            # 解析收益消息
            try:
                # 尝试解析JSON格式
                data = json.loads(content)
                
                # 输出解析后的数据
                self.starfire_log(f"🔍 [DEBUG] 解析后数据类型: {type(data)}", "purple")
                self.starfire_log(f"🔍 [DEBUG] 数据内容: {data}", "purple")
                
                # 支持Go语言发送的格式 - 优先检查是否有total_income字段
                if 'total_income' in data:
                    # 新格式: 包含total_income字段
                    amount = float(data.get('amount', 0))
                    total = float(data.get('total_income', 0))
                    model = data.get('model', '')
                    usage = data.get('usage', {})
                    currency = data.get('currency', '¥')
                    
                    # 调试日志
                    self.starfire_log(f"🔍 解析收益: amount={amount}, total_income={total}", "gray")
                    
                    # 更新累计收益(直接使用服务端传来的total_income)
                    self.total_income = total
                    
                    # 显示toast通知
                    self.show_income_toast(amount, currency, model, usage)
                    
                    # 记录日志
                    log_msg = f"💰 收益到账: {amount:.6f} {currency}"
                    if model:
                        log_msg += f" (模型: {model})"
                    if usage:
                        tokens = usage.get('total_tokens', 0)
                        if tokens:
                            log_msg += f" [tokens: {tokens}]"
                    self.starfire_log(log_msg, "green")
                    self.starfire_log(f"📊 累计收益: {self.total_income:.6f} {currency}", "blue")
                    
                # 兼容旧格式: 只有type和amount,没有total_income
                elif 'type' in data and data['type'] == 'income' and 'total_income' not in data:
                    amount = data.get('amount', '0')
                    currency = data.get('currency', '¥')
                    message = data.get('message', '')
                    
                    # 更新累计收益
                    self.total_income += float(amount)
                    
                    # 显示toast通知
                    self.show_income_toast(amount, currency)
                    
                    # 记录日志
                    log_msg = f"💰 收益到账: {amount} {currency}"
                    if message:
                        log_msg += f" ({message})"
                    self.starfire_log(log_msg, "green")
                else:
                    # 其他类型的消息
                    self.starfire_log(f"📨 收到消息: {content}", "blue")
            except json.JSONDecodeError:
                # 不是JSON格式,尝试文本解析
                is_income, amount, currency = parse_income_message(content)
                if is_income:
                    self.total_income += float(amount)
                    self.show_income_toast(amount, currency)
                    self.starfire_log(f"💰 收益到账: {amount} {currency}", "green")
                else:
                    self.starfire_log(f"📨 {content}", "blue")
    
    def show_income_toast(self, amount, currency, model='', usage=None):
        """显示收益通知"""
        # 格式化金额显示
        if isinstance(amount, (int, float)):
            amount_str = f"{amount:.6f}" if amount < 0.01 else f"{amount:.2f}"
        else:
            amount_str = str(amount)
        
        # 构建消息内容
        message_lines = []
        
        # 模型信息放在最前面(最醒目)
        if model:
            message_lines.append(f"🤖 模型: {model}")
            message_lines.append(f"━━━━━━━━━━━━━━")
        
        message_lines.append(f"💵 本次收益: {amount_str} {currency}")
        message_lines.append(f"💰 累计总收益: {self.total_income:.6f} {currency}")
        
        # 添加token使用信息
        if usage and isinstance(usage, dict):
            prompt_tokens = usage.get('prompt_tokens', 0)
            completion_tokens = usage.get('completion_tokens', 0)
            total_tokens = usage.get('total_tokens', 0)
            if total_tokens:
                message_lines.append(f"📝 Tokens: ↑{prompt_tokens} ↓{completion_tokens}")
        
        message = "\n".join(message_lines)
        
        ToastNotification(
            self.root,
            message=message,
            title="💰 收益到账",
            duration=5000,
            toast_type="money"
        )
    
    def test_toast_notification(self):
        """测试Toast通知效果"""
        import random
        
        # 模拟不同类型的收益
        test_types = [
            ("15.80", "¥"),
            ("23.50", "¥"),
            ("8.20", "¥"),
            ("42.00", "¥")
        ]
        
        amount, currency = random.choice(test_types)
        self.total_income += float(amount)
        self.show_income_toast(amount, currency)
        
        # 同时在日志中显示
        self.starfire_log(f"✓ 测试收益通知: {amount} {currency} (累计: {self.total_income:.2f} {currency})", "green")
    
    def _reset_starfire_ui(self):
        self.start_starfire_btn.config(state=tk.NORMAL)
        self.stop_starfire_btn.config(state=tk.DISABLED)
        self.starfire_status_label.config(
            text=" ● 未运行 ",
            bg="#D3D3D3",
            fg="gray"
        )
        self.starfire_running = False
    
    def check_ollama(self):
        """检查Ollama是否已安装 - 关键修复：添加 CREATE_NO_WINDOW"""
        # 只在 ollama 模式下检查
        if self.model_mode_var.get() != 'ollama':
            return
        
        try:
            result = subprocess.run(
                ["ollama", "--version"], 
                capture_output=True, 
                text=True, 
                timeout=5,
                creationflags=SUBPROCESS_FLAGS  # ← 关键修复
            )
            
            if result.returncode == 0:
                version = result.stdout.strip()
                self.status_label.config(
                    text=f"✓ Ollama 已安装 ({version})", 
                    foreground="green"
                )
                self.log(f"检测到 Ollama: {version}", "green")
                self.load_models()
            else:
                self.show_install_prompt()
        except FileNotFoundError:
            self.show_install_prompt()
        except Exception as e:
            self.status_label.config(
                text=f"✗ 检查失败: {str(e)}", 
                foreground="red"
            )
            self.log(f"错误: {str(e)}", "red")
    
    def show_install_prompt(self):
        # 只在ollama模式下显示安装提示
        if self.model_mode_var.get() != 'ollama':
            return
        
        self.status_label.config(
            text="✗ 未检测到 Ollama", 
            foreground="red"
        )
        self.log("未检测到 Ollama 安装", "red")
        
        response = messagebox.askyesno(
            "Ollama 未安装",
            "未检测到 Ollama 安装。\n\n是否前往官网下载安装？"
        )
        
        if response:
            webbrowser.open("https://ollama.com/download")
            self.log("已打开 Ollama 官网")
    
    def check_running_models(self):
        """检查正在运行的模型 - 关键修复：添加 CREATE_NO_WINDOW"""
        # 只在 ollama 模式下检查
        if self.model_mode_var.get() != 'ollama':
            self.root.after(5000, self.check_running_models)
            return
        
        try:
            result = subprocess.run(
                ["ollama", "ps"],
                capture_output=True,
                text=True,
                timeout=5,
                creationflags=SUBPROCESS_FLAGS  # ← 关键修复
            )
            
            if result.returncode == 0:
                lines = result.stdout.strip().split('\n')
                old_running = self.running_models.copy()
                self.running_models.clear()
                
                for line in lines[1:]:
                    parts = line.split()
                    if parts:
                        model_name = parts[0]
                        self.running_models.add(model_name)
                
                if old_running != self.running_models:
                    self.update_model_colors()
                    self.update_running_label()
        except:
            pass
        
        # 每 5 秒检查一次（降低频率）
        self.root.after(5000, self.check_running_models)
    
    def update_running_label(self):
        if self.running_models:
            running_list = ", ".join(list(self.running_models)[:2])
            if len(self.running_models) > 2:
                running_list += f" +{len(self.running_models)-2}"
            self.running_label.config(text=f"● {running_list}")
        else:
            self.running_label.config(text="")
    
    def update_model_colors(self):
        for item in self.model_tree.get_children():
            values = self.model_tree.item(item)['values']
            if len(values) >= 2:
                model_name = values[1]
                
                if model_name in self.running_models:
                    self.model_tree.item(item, tags=('running',))
                else:
                    self.model_tree.item(item, tags=('idle',))
        
        self.model_tree.tag_configure('running', background='#90EE90', foreground='darkgreen')
        self.model_tree.tag_configure('idle', background='#D3D3D3', foreground='gray')
    
    def load_models(self):
        """加载已安装的模型列表 - 关键修复：添加 CREATE_NO_WINDOW"""
        try:
            for item in self.model_tree.get_children():
                self.model_tree.delete(item)
            
            self.log("正在获取模型列表...")
            
            result = subprocess.run(
                ["ollama", "list"], 
                capture_output=True, 
                text=True, 
                timeout=10,
                creationflags=SUBPROCESS_FLAGS  # ← 关键修复
            )
            
            if result.returncode == 0:
                lines = result.stdout.strip().split('\n')
                
                if len(lines) <= 1:
                    self.log("未找到已安装的模型", "orange")
                    messagebox.showinfo("提示", "未找到已安装的模型\n请先使用 'ollama pull <model>' 下载模型")
                    return
                
                category_count = {}
                
                for line in lines[1:]:
                    parts = line.split()
                    if len(parts) >= 3:
                        name = parts[0]
                        size = parts[1] if len(parts) > 1 else "N/A"
                        modified = " ".join(parts[2:]) if len(parts) > 2 else "N/A"
                        
                        category = self.get_model_category(name)
                        icon = self.get_category_icon(category)
                        category_name = self.get_category_name(category)
                        category_display = f"{icon} {category_name}"
                        
                        category_count[category] = category_count.get(category, 0) + 1
                        
                        self.model_tree.insert(
                            "", 
                            tk.END, 
                            values=(category_display, name, size, modified)
                        )
                
                self.update_model_colors()
                self.update_running_label()
                
                total = len(lines) - 1
                category_info = ", ".join([f"{self.get_category_name(cat)}: {count}" for cat, count in category_count.items()])
                self.log(f"成功加载 {total} 个模型 ({category_info})", "green")
                
                self.run_btn.config(state=tk.NORMAL)
                if self.running_models:
                    self.stop_btn.config(state=tk.NORMAL)
            else:
                error_msg = result.stderr.strip()
                self.log(f"获取模型列表失败: {error_msg}", "red")
                messagebox.showerror("错误", f"获取模型列表失败:\n{error_msg}")
        
        except Exception as e:
            self.log(f"加载模型列表时出错: {str(e)}", "red")
            messagebox.showerror("错误", f"加载模型列表失败:\n{str(e)}")
    
    def run_model(self):
        selection = self.model_tree.selection()
        
        if not selection:
            messagebox.showwarning("提示", "请先选择一个模型")
            return
        
        item = self.model_tree.item(selection[0])
        model_name = item['values'][1]
        category = item['values'][0]
        
        if model_name in self.running_models:
            messagebox.showinfo("提示", f"模型 {model_name} 已经在运行中")
            return
        
        self.log(f"\n{'='*50}", "blue")
        self.log(f"正在启动: {model_name} [{category}]", "blue")
        self.log(f"{'='*50}\n", "blue")
        
        threading.Thread(target=self._run_model_thread, args=(model_name,), daemon=True).start()
    
    def _run_model_thread(self, model_name):
        """在后台线程中运行模型 - 关键修复：添加 CREATE_NO_WINDOW"""
        try:
            if platform.system() == "Windows":
                process = subprocess.Popen(
                    ["ollama", "run", "--keepalive", "24h", model_name],
                    stdin=subprocess.PIPE,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.PIPE,
                    text=True,
                    creationflags=SUBPROCESS_FLAGS  # ← 关键修复
                )
            else:
                process = subprocess.Popen(
                    ["ollama", "run", "--keepalive", "24h", model_name],
                    stdin=subprocess.PIPE,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.PIPE,
                    text=True
                )
            
            try:
                process.stdin.write("/bye\n")
                process.stdin.flush()
                process.stdin.close()
                
                process.wait(timeout=10)
                
                self.log(f"✓ 模型 {model_name} 已启动 (保持24h)", "green")
                
                self.running_models.add(model_name)
                self.root.after(100, self.update_model_colors)
                self.root.after(100, self.update_running_label)
                self.root.after(100, lambda: self.stop_btn.config(state=tk.NORMAL))
                
            except subprocess.TimeoutExpired:
                process.kill()
                self.log(f"✗ 启动模型超时", "red")
            
        except Exception as e:
            self.log(f"✗ 运行模型时出错: {str(e)}", "red")
    
    def stop_model(self):
        """停止选中的模型 - 关键修复：添加 CREATE_NO_WINDOW"""
        selection = self.model_tree.selection()
        
        if not selection:
            messagebox.showwarning("提示", "请先选择一个模型")
            return
        
        item = self.model_tree.item(selection[0])
        model_name = item['values'][1]
        
        if model_name not in self.running_models:
            messagebox.showinfo("提示", f"模型 {model_name} 未在运行中")
            return
        
        try:
            self.log(f"\n正在停止: {model_name}...", "orange")
            
            result = subprocess.run(
                ["ollama", "stop", model_name],
                capture_output=True,
                text=True,
                timeout=10,
                creationflags=SUBPROCESS_FLAGS  # ← 关键修复
            )
            
            if result.returncode == 0:
                self.log(f"✓ 模型 {model_name} 已停止", "green")
                
                self.running_models.discard(model_name)
                self.update_model_colors()
                self.update_running_label()
                
                if not self.running_models:
                    self.stop_btn.config(state=tk.DISABLED)
            else:
                error_msg = result.stderr.strip()
                self.log(f"✗ 停止模型失败: {error_msg}", "red")
                
        except subprocess.TimeoutExpired:
            self.log(f"✗ 停止模型超时", "red")
        except Exception as e:
            self.log(f"✗ 停止模型时出错: {str(e)}", "red")
    
    def open_starfire(self):
        url = "http://115.190.26.60/"
        webbrowser.open(url)
        self.starfire_log(f"已打开 Starfire 官网: {url}")


def main():
    """主函数 - 优化启动画面"""
    splash = SplashScreen()
    
    splash.update_status("正在初始化...")
    splash.root.after(300)
    splash.root.update()
    
    splash.update_status("正在加载组件...")
    splash.root.after(300)
    splash.root.update()
    
    splash.update_status("准备就绪...")
    splash.root.after(200)
    splash.root.update()
    
    splash.close()
    
    root = tk.Tk()
    app = StarFireAPP(root)
    root.mainloop()


if __name__ == "__main__":
    main()