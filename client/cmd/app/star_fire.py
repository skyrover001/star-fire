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
import traceback
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

def validate_url(url, require_path=False):
    """验证URL格式
    必须以 http:// 或 https:// 开头
    支持带路径的URL，如 http://example.com/v1, http://example.com:8080/chat/v1/
    返回: (is_valid, error_message)
    """
    if not url:
        return False, "URL不能为空"
    
    url = url.strip()
    
    if not url.startswith('http://') and not url.startswith('https://'):
        return False, "URL必须以 http:// 或 https:// 开头"
    
    # 提取 host 部分用于验证
    if url.startswith('https://'):
        host_part = url[8:]
    else:
        host_part = url[7:]
    
    # 去掉路径部分，获取 host:port
    if '/' in host_part:
        host_part = host_part.split('/')[0]
    
    # 检查 host 部分是否为空
    if not host_part:
        return False, "URL格式错误，缺少主机名"
    
    # 检查是否包含端口
    if ':' in host_part:
        try:
            port = int(host_part.split(':')[1])
            if port < 1 or port > 65535:
                return False, "端口号必须在 1-65535 之间"
        except ValueError:
            return False, "端口号格式错误"
    
    # 如果需要路径但没有路径
    if require_path and '/' not in url[url.find('://') + 3:]:
        return False, "URL需要包含路径，如 /v1"
    
    return True, ""

def validate_host(host):
    """验证服务器地址格式（简化版，用于服务器地址）
    必须以 http:// 或 https:// 开头
    返回: (is_valid, error_message)
    """
    return validate_url(host, require_path=False)

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
        self.clients = []  # 存储已连接的客户端套接字
        self.clients_lock = threading.Lock()  # 客户端列表锁
        
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
        # 添加到客户端列表
        with self.clients_lock:
            self.clients.append(client_socket)
        
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
            # 从客户端列表移除
            with self.clients_lock:
                if client_socket in self.clients:
                    self.clients.remove(client_socket)
            
            client_socket.close()
            if self.callback:
                self.callback('disconnect', f"客户端断开: {client_address}")
    
    def stop(self):
        """停止TCP服务器"""
        self.running = False
        
        # 关闭所有客户端连接
        with self.clients_lock:
            for client in self.clients:
                try:
                    client.close()
                except:
                    pass
            self.clients.clear()
        
        if self.server_socket:
            try:
                self.server_socket.close()
            except:
                pass
        return True, "TCP服务器已停止"
    
    def send_to_all_clients(self, message):
        """向所有已连接的客户端发送消息"""
        if not isinstance(message, str):
            message = json.dumps(message, ensure_ascii=False)
        
        try:
            message_bytes = message.encode('utf-8')
            message_length = len(message_bytes)
            length_prefix = struct.pack('!I', message_length)
            full_message = length_prefix + message_bytes
            
            if self.callback:
                self.callback('error', f"📤 准备发送消息: 长度={message_length} 字节")
            
            failed_clients = []
            with self.clients_lock:
                for client in self.clients:
                    try:
                        client.sendall(full_message)
                        if self.callback:
                            self.callback('error', f"✓ 已发送到客户端: {client.getpeername()}")
                    except Exception as e:
                        failed_clients.append((client, e))
                        if self.callback:
                            self.callback('error', f"✗ 发送失败: {str(e)}")
            
            # 移除发送失败的客户端
            if failed_clients:
                with self.clients_lock:
                    for client, error in failed_clients:
                        if client in self.clients:
                            self.clients.remove(client)
                        try:
                            client.close()
                        except:
                            pass
                        if self.callback:
                            self.callback('error', f"发送消息失败: {str(error)}")
            
            return len(self.clients) - len(failed_clients)  # 返回成功发送的客户端数量
            
        except Exception as e:
            if self.callback:
                self.callback('error', f"❌ send_to_all_clients 异常: {str(e)}")
            return 0


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
        self.pending_price_message = None  # 待发送的价格配置消息
        
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
        
        # 初始化模型价格窗口引用
        self.model_price_window = None
        self.model_price_tree = None
        
        self.create_widgets()
        self.check_ollama()
        self.check_running_models()
    
    def load_config(self):
        self.config = {
            'host': 'http://111.228.58.164',
            'username': '',
            'password': '',
            'jwt_token': '',  # JWT token
            'ippm': '3.8',   # 默认输入价格 3.8 元 / 百万 tokens
            'oppm': '8.3',   # 默认输出价格 8.3 元 / 百万 tokens
            'model_mode': 'ollama',  # ollama, vllm, proxy, llamacpp
            'proxy_base_url': 'http://localhost:8000/v1',
            'proxy_api_key': '',
            'ollama_num_parallel': '',  # Ollama并发请求数
            'model_prices': {}  # 每个模型的价格配置 {model_name: {ippm: xx, oppm: xx, cippm: xx}}
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
        # 获取当前输入框的值 - 移除了 ippm 和 oppm，它们现在在模型价格设置中
        current_values = {
            'host': self.host_entry.get().strip(),
            'username': self.username_entry.get().strip(),
            'password': self.password_entry.get().strip(),
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
            # 在代理模式下允许刷新以显示代理模型，但禁用运行/停止
            self.refresh_btn.config(state=tk.NORMAL)
            self.run_btn.config(state=tk.DISABLED)
            self.stop_btn.config(state=tk.DISABLED)
            self.log("已切换到代理模式", "blue")
            # 清空本地运行状态，避免代理模型显示为运行中
            self.running_models.clear()
            self.update_model_colors()
            self.update_running_label()
            # 立即加载代理模型到左侧列表
            try:
                self.load_models()
            except Exception as e:
                self.log(f"加载代理模型失败: {str(e)}", "red")
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
        # 切换模型接入方式时自动刷新模型价格窗口
        if hasattr(self, 'model_price_window') and self.model_price_window:
            try:
                if hasattr(self, 'model_price_tree') and self.model_price_tree:
                    self.starfire_log("🔄 模式切换，正在刷新模型价格列表...", "blue")
                    self.refresh_model_price_list(self.model_price_tree, self.model_price_window)
                else:
                    self.starfire_log("⚠️ 模型价格窗口存在，但列表未初始化", "orange")
            except Exception as e:
                self.starfire_log(f"❌ 刷新模型价格列表失败: {str(e)}", "red")
                try:
                    if self.model_price_window.winfo_exists() and hasattr(self, 'model_price_tree') and self.model_price_tree:
                        self.refresh_model_price_list(self.model_price_tree, self.model_price_window)
                except Exception as e2:
                    self.starfire_log(f"自动刷新模型价格窗口失败: {str(e2)}", "red")
    
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
        def validate_proxy_url_on_blur():
            """Base URL失焦验证"""
            url = self.proxy_base_url_entry.get().strip()
            if url:
                is_valid, err_msg = validate_url(url)
                if not is_valid:
                    self.starfire_log(f"⚠️ Base URL格式警告: {err_msg}", "orange")
        
        self.proxy_base_url_entry = ttk.Entry(proxy_url_frame)
        self.proxy_base_url_entry.insert(0, self.config.get('proxy_base_url', 'http://localhost:8000/v1'))
        self.proxy_base_url_entry.pack(side=tk.LEFT, fill=tk.X, expand=True, padx=(5, 0))
        self.proxy_base_url_entry.bind('<FocusOut>', lambda e: (self.auto_save_config('proxy_base_url'), validate_proxy_url_on_blur()))
        
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
        
        # 用户名
        username_frame = ttk.Frame(config_frame)
        username_frame.pack(fill=tk.X, pady=5)
        ttk.Label(username_frame, text="用户名:", width=12).pack(side=tk.LEFT)
        self.username_entry = ttk.Entry(username_frame)
        self.username_entry.insert(0, self.config.get('username', ''))
        self.username_entry.pack(side=tk.LEFT, fill=tk.X, expand=True, padx=(5, 0))
        self.username_entry.bind('<FocusOut>', lambda e: self.auto_save_config('username'))
        
        # 密码
        password_frame = ttk.Frame(config_frame)
        password_frame.pack(fill=tk.X, pady=5)
        ttk.Label(password_frame, text="密码:", width=12).pack(side=tk.LEFT)
        self.password_entry = ttk.Entry(password_frame, show="*")
        self.password_entry.insert(0, self.config.get('password', ''))
        self.password_entry.pack(side=tk.LEFT, fill=tk.X, expand=True, padx=(5, 0))
        self.password_entry.bind('<FocusOut>', lambda e: self.auto_save_config('password'))
        
        def toggle_password():
            if self.password_entry['show'] == '*':
                self.password_entry['show'] = ''
                toggle_pwd_btn.config(text="👁️")
            else:
                self.password_entry['show'] = '*'
                toggle_pwd_btn.config(text="🔒")
        
        toggle_pwd_btn = ttk.Button(password_frame, text="🔒", width=3, command=toggle_password)
        toggle_pwd_btn.pack(side=tk.LEFT, padx=(5, 0))
        
        # 登录状态显示
        login_status_frame = ttk.Frame(config_frame)
        login_status_frame.pack(fill=tk.X, pady=5)
        ttk.Label(login_status_frame, text="登录状态:", width=12).pack(side=tk.LEFT)
        self.login_status_label = tk.Label(
            login_status_frame,
            text=" ○ 未登录 ",
            bg="#D3D3D3",
            fg="gray",
            relief=tk.RAISED,
            padx=10,
            font=("Arial", 9)
        )
        self.login_status_label.pack(side=tk.LEFT, padx=(5, 0))
        
        # 去掉获取Token按钮，仅保留显示与显隐切换
        # 添加收益信息展示
        income_frame = ttk.Frame(config_frame)
        income_frame.pack(fill=tk.X, pady=5)
        ttk.Label(income_frame, text="总收益:", width=12).pack(side=tk.LEFT)
        self.total_income_label = ttk.Label(
            income_frame,
            text="0.00 ¥",
            foreground="green",
            font=("Arial", 10, "bold")
        )
        self.total_income_label.pack(side=tk.LEFT, padx=(5, 0))

        latest_frame = ttk.Frame(config_frame)
        latest_frame.pack(fill=tk.X, pady=5)
        ttk.Label(latest_frame, text="最新收益:", width=12).pack(side=tk.LEFT)
        self.latest_income_label = ttk.Label(
            latest_frame,
            text="0.00 ¥",
            foreground="blue",
            font=("Arial", 10)
        )
        self.latest_income_label.pack(side=tk.LEFT, padx=(5, 0))
        
        # 模型价格设置按钮
        model_price_frame = ttk.Frame(config_frame)
        model_price_frame.pack(fill=tk.X, pady=(10, 0))
        
        ttk.Button(
            model_price_frame,
            text="📋 模型价格设置",
            command=self.open_model_price_window,
            width=20
        ).pack(side=tk.LEFT, padx=5)
        
        ttk.Label(
            model_price_frame,
            text="为每个模型单独设置价格",
            foreground="gray",
            font=("Arial", 8)
        ).pack(side=tk.LEFT, padx=5)
        
        starfire_button_frame = ttk.Frame(config_frame)
        starfire_button_frame.pack(fill=tk.X, pady=(10, 0))
        
        self.login_btn = ttk.Button(
            starfire_button_frame,
            text="🔐 登录",
            command=self.login_to_server,
            width=15
        )
        self.login_btn.pack(side=tk.LEFT, padx=5)
        
        self.save_config_btn = ttk.Button(
            starfire_button_frame,
            text="💾 保存配置",
            command=self.save_config_action,
            width=15
        )
        self.save_config_btn.pack(side=tk.LEFT, padx=5)
        
        self.fetch_income_btn = ttk.Button(
            starfire_button_frame,
            text="💰 刷新收益",
            command=self.fetch_income_data,
            width=15,
            state=tk.DISABLED
        )
        self.fetch_income_btn.pack(side=tk.LEFT, padx=5)
        
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

        # 启动时若默认为代理模式且配置完整，则立即加载代理模型列表
        try:
            initial_mode = self.model_mode_var.get()
            if initial_mode == 'proxy':
                base_url = self.proxy_base_url_entry.get().strip()
                api_key = self.proxy_api_key_entry.get().strip()
                if base_url and api_key:
                    self.log("启动为代理模式，正在加载模型列表...", "blue")
                    # 确保按钮状态正确
                    self.refresh_btn.config(state=tk.NORMAL)
                    self.run_btn.config(state=tk.DISABLED)
                    self.stop_btn.config(state=tk.DISABLED)
                    self.load_models()
        except Exception as e:
            self.log(f"启动时加载代理模型失败: {str(e)}", "red")
    
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
        self.config['username'] = self.username_entry.get().strip()
        self.config['password'] = self.password_entry.get().strip()
        self.config['model_mode'] = self.model_mode_var.get()
        self.config['proxy_base_url'] = self.proxy_base_url_entry.get().strip()
        self.config['proxy_api_key'] = self.proxy_api_key_entry.get().strip()
        self.config['ollama_num_parallel'] = self.ollama_num_parallel_entry.get().strip()
        
        self.save_config()
        self.starfire_log("✓ 配置已保存", "green")
        messagebox.showinfo("成功", "配置已保存！")
    
    def login_to_server(self):
        """登录到服务器获取JWT token"""
        host = self.host_entry.get().strip()
        username = self.username_entry.get().strip()
        password = self.password_entry.get().strip()
        
        is_valid, err_msg = validate_host(host)
        if not is_valid:
            messagebox.showwarning("提示", f"服务器地址格式错误: {err_msg}\n\n正确格式示例:\n• http://111.228.58.164\n• https://chat.example.com\n• http://123.12.1.123:8080")
            return
        
        if not all([host, username, password]):
            messagebox.showwarning("提示", "请填写服务器地址、用户名和密码！")
            return
        
        # 创建本地验证对话框
        captcha_window = tk.Toplevel(self.root)
        captcha_window.title("安全验证")
        captcha_window.transient(self.root)
        captcha_window.grab_set()
        captcha_window.resizable(False, False)
        
        # 居中显示
        win_w, win_h = 360, 180
        captcha_window.withdraw()
        captcha_window.update_idletasks()
        sx = (captcha_window.winfo_screenwidth() - win_w) // 2
        sy = (captcha_window.winfo_screenheight() - win_h) // 2
        captcha_window.geometry(f"{win_w}x{win_h}+{sx}+{sy}")
        captcha_window.deiconify()
        
        frame = ttk.Frame(captcha_window, padding="15 12")
        frame.pack(fill=tk.BOTH, expand=True)
        
        ttk.Label(frame, text="🔐 安全验证", font=("Arial", 11, "bold")).pack(pady=(0, 8))
        
        # 生成随机算术题作为验证码（加减乘除，100以内）
        import random
        operations = [
            ('+', lambda a, b: a + b),
            ('-', lambda a, b: a - b),
            ('×', lambda a, b: a * b),
            ('÷', lambda a, b: a // b if b != 0 and a % b == 0 else None)
        ]
        
        def generate_question():
            while True:
                op_symbol, op_func = random.choice(operations)
                if op_symbol == '÷':
                    # 除法：确保能整除
                    divisor = random.randint(2, 10)
                    quotient = random.randint(2, 10)
                    num1 = divisor * quotient
                    num2 = divisor
                elif op_symbol == '-':
                    # 减法：确保结果为正数
                    num1 = random.randint(10, 99)
                    num2 = random.randint(1, num1)
                else:
                    # 加法和乘法
                    if op_symbol == '×':
                        num1 = random.randint(2, 12)
                        num2 = random.randint(2, 12)
                    else:
                        num1 = random.randint(1, 99)
                        num2 = random.randint(1, 99)
                
                result = op_func(num1, num2)
                if result is not None and 0 <= result <= 100:
                    return num1, num2, op_symbol, result
        
        num1, num2, op_symbol, correct_answer = generate_question()
        
        # 算式 + 输入框 + 换一题 在同一行
        qa_frame = ttk.Frame(frame)
        qa_frame.pack(pady=8)
        
        question_label = tk.Label(
            qa_frame,
            text=f"{num1} {op_symbol} {num2} = ",
            font=("Arial", 15, "bold"),
            fg="#2c3e50"
        )
        question_label.pack(side=tk.LEFT)
        
        answer_entry = ttk.Entry(qa_frame, font=("Arial", 13), width=6, justify=tk.CENTER)
        answer_entry.pack(side=tk.LEFT, padx=(2, 6))
        answer_entry.focus()
        
        def refresh_captcha():
            """刷新验证码"""
            nonlocal num1, num2, op_symbol, correct_answer
            num1, num2, op_symbol, correct_answer = generate_question()
            question_label.config(text=f"{num1} {op_symbol} {num2} = ")
            answer_entry.delete(0, tk.END)
            answer_entry.focus()
        
        ttk.Button(qa_frame, text="换一题", command=refresh_captcha, width=6).pack(side=tk.LEFT)
        
        def do_login():
            """执行登录"""
            # 验证答案
            try:
                user_answer = int(answer_entry.get().strip())
            except ValueError:
                messagebox.showwarning("提示", "请输入有效的数字！", parent=captcha_window)
                return
            
            # 验证算术题答案
            if user_answer != correct_answer:
                messagebox.showerror("错误", "验证码错误，请重试！", parent=captcha_window)
                refresh_captcha()
                return
            
            captcha_window.destroy()
            
            def _login():
                try:
                    import urllib.request
                    import urllib.parse
                    
                    base_url = f"http://{host}" if not host.startswith('http') else host
                    login_url = f"{base_url}/api/login"
                    
                    login_data = {
                        'username': username,
                        'password': password,
                        'captcha': True
                    }
                    
                    data = json.dumps(login_data).encode('utf-8')
                    req = urllib.request.Request(login_url, data=data, method='POST')
                    req.add_header('Content-Type', 'application/json')
                    
                    with urllib.request.urlopen(req, timeout=10) as response:
                        result = json.loads(response.read().decode('utf-8'))
                        
                        if response.status == 200 or response.status == 201:
                            jwt_token = result['token']
                            self.config['jwt_token'] = jwt_token
                            self.save_config()
                            
                            def _update_ui():
                                self.login_status_label.config(
                                    text=" ● 已登录 ",
                                    bg="#90EE90",
                                    fg="darkgreen"
                                )
                                self.fetch_income_btn.config(state=tk.NORMAL)
                                self.starfire_log(f"✓ 登录成功！", "green")
                                messagebox.showinfo("成功", "登录成功！")
                                # 自动获取收益
                                self.fetch_income_data()
                            
                            self.root.after(0, _update_ui)
                        else:
                            print("result:", result)
                            error_msg = result.get('message', '登录失败')
                            self.root.after(0, lambda: self.starfire_log(f"❌ {error_msg}", "red"))
                            self.root.after(0, lambda: messagebox.showerror("错误", error_msg))
                            
                except Exception as e:
                    error_msg = f"登录失败: {str(e)}"
                    self.root.after(0, lambda: self.starfire_log(f"❌ {error_msg}", "red"))
                    self.root.after(0, lambda: messagebox.showerror("错误", error_msg))
            
            threading.Thread(target=_login, daemon=True).start()
        
        btn_frame = ttk.Frame(frame)
        btn_frame.pack(pady=(10, 0))
        
        ttk.Button(btn_frame, text="✓ 登录", command=do_login, width=10).pack(side=tk.LEFT, padx=8)
        ttk.Button(btn_frame, text="✗ 取消", command=captcha_window.destroy, width=10).pack(side=tk.LEFT, padx=8)
        
        answer_entry.bind('<Return>', lambda e: do_login())
    
    def fetch_income_data(self):
        """获取用户收益数据"""
        jwt_token = self.config.get('jwt_token', '')
        host = self.host_entry.get().strip()
        
        if not jwt_token:
            messagebox.showwarning("提示", "请先登录！")
            return
        
        is_valid, err_msg = validate_host(host)
        if not is_valid:
            messagebox.showwarning("提示", f"服务器地址格式错误: {err_msg}\n\n正确格式示例:\n• http://111.228.58.164\n• https://chat.example.com\n• http://123.12.1.123:8080")
            return
        
        def _fetch():
            try:
                import urllib.request
                
                base_url = f"http://{host}" if not host.startswith('http') else host
                income_url = f"{base_url}/api/user/income"
                
                req = urllib.request.Request(income_url)
                req.add_header('Authorization', f'Bearer {jwt_token}')
                
                with urllib.request.urlopen(req, timeout=10) as response:
                    result = json.loads(response.read().decode('utf-8'))
                    
                    if response.status == 201 or response.status == 200:
                        data_list = result.get('data', [])
                        
                        # 计算总收益: sum(((input - cached) * ippm + cached * cippm + oppm * output) / 1000000)
                        total_revenue = 0.0
                        for item in data_list:
                            ippm = float(item.get('IPPM', 0))
                            oppm = float(item.get('OPPM', 0))
                            cippm = float(item.get('CIPPM', 0))
                            input_tokens = int(item.get('InputTokens', 0))
                            output_tokens = int(item.get('OutputTokens', 0))
                            cached_tokens = int(item.get('CachedTokens', 0))
                            
                            non_cached = input_tokens - cached_tokens
                            revenue = (non_cached * ippm + cached_tokens * cippm + oppm * output_tokens) / 1000000
                            total_revenue += revenue
                        
                        # 更新UI
                        def _update():
                            self.total_income = total_revenue
                            self.total_income_label.config(text=f"{total_revenue:.6f} ¥")
                            
                            if data_list:
                                latest = data_list[0]
                                latest_ippm = float(latest.get('IPPM', 0))
                                latest_oppm = float(latest.get('OPPM', 0))
                                latest_cippm = float(latest.get('CIPPM', 0))
                                latest_input = int(latest.get('InputTokens', 0))
                                latest_output = int(latest.get('OutputTokens', 0))
                                latest_cached = int(latest.get('CachedTokens', 0))
                                latest_non_cached = latest_input - latest_cached
                                latest_revenue = (latest_non_cached * latest_ippm + latest_cached * latest_cippm + latest_oppm * latest_output) / 1000000
                                self.latest_income_label.config(text=f"{latest_revenue:.6f} ¥")
                            
                            self.starfire_log(f"✓ 已刷新收益数据，总收益: {total_revenue:.6f} ¥ ({len(data_list)} 条记录)", "green")
                        
                        self.root.after(0, _update)
                    else:
                        print("result:", result)
                        error_msg = result.get('message', '获取收益失败')
                        self.root.after(0, lambda: self.starfire_log(f"❌ {error_msg}", "red"))
                        
            except Exception as e:
                error_msg = f"获取收益失败: {str(e)}"
                self.root.after(0, lambda: self.starfire_log(f"❌ {error_msg}", "red"))
        
        threading.Thread(target=_fetch, daemon=True).start()
    
    def get_register_token(self):
        """从服务器获取注册token"""
        jwt_token = self.config.get('jwt_token', '')
        host = self.host_entry.get().strip()
        
        if not jwt_token:
            self.starfire_log("❌ 未登录，无法获取注册token", "red")
            return None
        
        try:
            import urllib.request
            
            base_url = f"http://{host}" if not host.startswith('http') else host
            token_url = f"{base_url}/api/user/register-token"
            
            print("Fetching register token from:", token_url," with JWT:", jwt_token)
            req = urllib.request.Request(token_url,method="POST",data=b'')
            req.add_header('Authorization', f'Bearer {jwt_token}')
            
            with urllib.request.urlopen(req, timeout=10) as response:
                result = json.loads(response.read().decode('utf-8'))
                
                print("Register token response:", result)
                if response.status == 200 or response.status == 201:
                    token = result['token']
                    self.starfire_log(f"✓ 获取注册token成功", "green")
                    return token
                else:
                    error_msg = result.get('message', '获取token失败')
                    self.starfire_log(f"❌ {error_msg}", "red")
                    return None
                    
        except Exception as e:
            self.starfire_log(f"❌ 获取注册token失败: {str(e)}", "red")
            return None

    def get_all_available_models(self):
        """返回所有可用模型及其引擎类型的字典 {model_name: engine}"""
        models = {}
        
        # 获取ollama当前在线运行的模型（使用 ollama ps）
        try:
            result = subprocess.run(
                ["ollama", "ps"],
                capture_output=True,
                text=True,
                timeout=10,
                creationflags=SUBPROCESS_FLAGS
            )
            if result.returncode == 0:
                lines = result.stdout.strip().split('\n')
                for line in lines[1:]:
                    parts = line.split()
                    if parts:
                        models[parts[0]] = 'ollama'
        except Exception as e:
            self.starfire_log(f"❌ 获取Ollama模型列表失败: {str(e)}", "red")

        # 获取代理模型
        try:
            base_url = self.proxy_base_url_entry.get().strip()
            api_key = self.proxy_api_key_entry.get().strip()
            if base_url and api_key:
                base_url = self.proxy_base_url_entry.get().strip().rstrip('/')
                api_key = self.proxy_api_key_entry.get().strip()
                if base_url:
                    models_url = f"{base_url}/models"
                import urllib.request
                req = urllib.request.Request(models_url)
                req.add_header('Authorization', f'Bearer {api_key}')
                with urllib.request.urlopen(req, timeout=10) as response:
                    data = json.loads(response.read().decode('utf-8'))
                    if 'data' in data:
                        for m in data['data']:
                            models[m['id']] = 'openai'
                    elif 'models' in data:
                        for m in data['models']:
                            models[m] = 'openai'
        except Exception as e:
            self.starfire_log(f"❌ 获取代理模型失败: {str(e)}", "red")

        return models
    
    def open_model_price_window(self):
        """打开模型价格设置窗口"""
        # 创建新窗口
        price_window = tk.Toplevel(self.root)
        price_window.title("模型价格设置")
        price_window.transient(self.root)
        
        # 居中显示
        pw_w, pw_h = 720, 500
        price_window.withdraw()
        price_window.update_idletasks()
        px = (price_window.winfo_screenwidth() - pw_w) // 2
        py = (price_window.winfo_screenheight() - pw_h) // 2
        price_window.geometry(f"{pw_w}x{pw_h}+{px}+{py}")
        price_window.deiconify()

        # 记录窗口实例
        self.model_price_window = price_window
        
        # 顶部说明
        info_frame = ttk.Frame(price_window, padding="10")
        info_frame.pack(fill=tk.X)
        ttk.Label(
            info_frame,
            text=(
                "💡 为每个模型单独设置输入/输出价格；"
                "如未设置则使用默认价格 (输入 < 10 元/百万 tokens，输出 < 20 元/百万 tokens)。\n"
                "关闭窗口即自动保存并同步到 Starfire ，无需手动发送。"
            ),
            foreground="blue",
            font=("Arial", 9),
            wraplength=650
        ).pack(anchor=tk.W)
        
        # 按钮区域
        button_frame = ttk.Frame(price_window, padding="10")
        button_frame.pack(fill=tk.X)
        
        ttk.Button(
            button_frame,
            text="🔄 刷新模型列表",
            command=lambda: self.refresh_model_price_list(tree, price_window)
        ).pack(side=tk.LEFT, padx=5)
        
        ttk.Button(
            button_frame,
            text="💾 保存所有价格",
            command=lambda: self.save_all_model_prices(tree, price_window)
        ).pack(side=tk.LEFT, padx=5)
        
        # 模型列表区域
        list_frame = ttk.Frame(price_window, padding="10")
        list_frame.pack(fill=tk.BOTH, expand=True)
        
        # 创建表格
        columns = ("模型名称", "引擎", "输入价格(¥/M)", "输出价格(¥/M)", "缓存输入价格(¥/M)")
        tree = ttk.Treeview(list_frame, columns=columns, show="headings", height=15)
        
        tree.heading("模型名称", text="模型名称")
        tree.heading("引擎", text="引擎")
        tree.heading("输入价格(¥/M)", text="输入价格(¥/M)")
        tree.heading("输出价格(¥/M)", text="输出价格(¥/M)")
        tree.heading("缓存输入价格(¥/M)", text="缓存输入价格(¥/M)")
        
        tree.column("模型名称", width=250)
        tree.column("引擎", width=80, anchor=tk.CENTER)
        tree.column("输入价格(¥/M)", width=120, anchor=tk.CENTER)
        tree.column("输出价格(¥/M)", width=120, anchor=tk.CENTER)
        tree.column("缓存输入价格(¥/M)", width=140, anchor=tk.CENTER)
        
        scrollbar = ttk.Scrollbar(list_frame, orient=tk.VERTICAL, command=tree.yview)
        tree.configure(yscrollcommand=scrollbar.set)
        
        tree.pack(side=tk.LEFT, fill=tk.BOTH, expand=True)
        scrollbar.pack(side=tk.RIGHT, fill=tk.Y)
        
        # 单击编辑（直接在单元格内编辑）
        tree.bind('<Button-1>', lambda e: self.edit_model_price_inline(tree, e, price_window))
        
        # 记录表格引用
        self.model_price_tree = tree

        # 加载模型列表
        self.refresh_model_price_list(tree, price_window)

        # 关闭窗口时自动保存并发送
        def on_close():
            self.save_all_model_prices(tree, price_window, auto=True)
            self.model_price_tree = None
            self.model_price_window = None
            price_window.destroy()
        price_window.protocol("WM_DELETE_WINDOW", on_close)
    
    def refresh_model_price_list(self, tree, window):
        """刷新模型价格列表"""
        # 清空现有数据
        for item in tree.get_children():
            tree.delete(item)
        
        # 获取模型列表（字典：{model_name: engine}）
        models_dict = self.get_all_available_models()
        
        if not models_dict:
            messagebox.showwarning("提示", "未找到可用模型！", parent=window)
            return
        
        # 获取已保存的价格配置（确保是同一个引用，便于就地更新）
        model_prices = self.config.setdefault('model_prices', {})

        # 全局默认价格（确保为字符串，便于显示）
        # 强制使用 3.8 和 8.3 作为默认值，忽略配置文件中的旧全局设置
        default_ippm = '3.8'
        default_oppm = '8.3'
        default_cippm = '1.0'

        config_changed = False
        
        # 填充数据
        for model, engine in sorted(models_dict.items()):
            entry = model_prices.get(model, {})

            ippm = str(entry.get('ippm', default_ippm)) if entry else default_ippm
            oppm = str(entry.get('oppm', default_oppm)) if entry else default_oppm
            cippm = str(entry.get('cippm', default_cippm)) if entry else default_cippm

            if not ippm:
                ippm = default_ippm
            if not oppm:
                oppm = default_oppm
            if not cippm:
                cippm = default_cippm

            if entry:
                # 统一写回标准化后的数值
                if entry.get('ippm') != ippm:
                    entry['ippm'] = ippm
                    config_changed = True
                if entry.get('oppm') != oppm:
                    entry['oppm'] = oppm
                    config_changed = True
                if entry.get('cippm') != cippm:
                    entry['cippm'] = cippm
                    config_changed = True
                if entry.get('engine') != engine:
                    entry['engine'] = engine
                    config_changed = True
            else:
                model_prices[model] = {
                    'ippm': ippm,
                    'oppm': oppm,
                    'cippm': cippm,
                    'engine': engine
                }
                config_changed = True

            tree.insert("", tk.END, values=(model, str(engine), ippm, oppm, cippm))
        
        if config_changed:
            self.save_config()
            self.starfire_log(
                f"✓ 已同步模型价格默认值 (输入: {default_ippm}, 输出: {default_oppm}, 缓存输入: {default_cippm})",
                "green"
            )
        
        self.starfire_log(f"✓ 已加载 {len(models_dict)} 个模型的价格配置", "green")
    
    def edit_model_price_inline(self, tree, event, parent_window):
        """在单元格内直接编辑模型价格"""
        region = tree.identify("region", event.x, event.y)
        if region != "cell":
            return
        
        column = tree.identify_column(event.x)
        row_id = tree.identify_row(event.y)
        
        # 不允许编辑模型名称和引擎列
        if not row_id or column in ("#1", "#2"):
            return
        
        # 获取单元格位置
        bbox = tree.bbox(row_id, column)
        if not bbox:
            return
        
        # 获取当前值
        values = list(tree.item(row_id)['values'])
        col_index = int(column[1:]) - 1
        current_value = str(values[col_index])
        
        # 创建编辑输入框（直接覆盖在单元格上）
        edit_entry = ttk.Entry(tree, width=15)
        edit_entry.insert(0, current_value)
        edit_entry.select_range(0, tk.END)
        edit_entry.focus()
        
        # 将输入框放置在单元格位置，缩短宽度为确认按钮留空间
        btn_w = 24
        entry_w = bbox[2] - btn_w - 2
        edit_entry.place(x=bbox[0], y=bbox[1], width=max(entry_w, 40), height=bbox[3])
        
        # 确认按钮
        confirm_btn = tk.Button(
            tree, text="✓", font=("Arial", 9, "bold"),
            bg="#4CAF50", fg="white", relief=tk.FLAT,
            activebackground="#45a049", cursor="hand2",
            bd=0, padx=2, pady=0
        )
        confirm_btn.place(x=bbox[0] + max(entry_w, 40) + 2, y=bbox[1], width=btn_w, height=bbox[3])
        
        def save_edit(event=None):
            new_value = edit_entry.get().strip()
            try:
                # 验证是有效数字
                float(new_value)
                values[col_index] = new_value
                tree.item(row_id, values=values)
            except ValueError:
                messagebox.showerror("错误", "请输入有效的数字！", parent=parent_window)
                edit_entry.focus()
                return
            
            confirm_btn.destroy()
            edit_entry.destroy()
        
        def cancel_edit(event=None):
            confirm_btn.destroy()
            edit_entry.destroy()
        
        confirm_btn.config(command=save_edit)
        
        # 绑定事件
        edit_entry.bind('<Return>', save_edit)
        edit_entry.bind('<Escape>', cancel_edit)
    
    def edit_model_price(self, tree, event):
        """编辑模型价格（旧方法，保留以防万一）"""
        region = tree.identify("region", event.x, event.y)
        if region != "cell":
            return
        
        column = tree.identify_column(event.x)
        row_id = tree.identify_row(event.y)
        
        if not row_id or column in ("#1", "#2"):  # 不允许编辑模型名称和引擎
            return
        
        # 获取当前值
        values = list(tree.item(row_id)['values'])
        col_index = int(column[1:]) - 1
        current_value = values[col_index]
        
        # 创建编辑框
        bbox = tree.bbox(row_id, column)
        if not bbox:
            return
        
        edit_window = tk.Toplevel(self.root)
        edit_window.title("编辑价格")
        edit_window.geometry("300x150")
        edit_window.transient(self.root)
        
        frame = ttk.Frame(edit_window, padding="20")
        frame.pack(fill=tk.BOTH, expand=True)
        
        model_name = values[0]
        price_type = "输入价格" if col_index == 2 else "输出价格"
        
        ttk.Label(frame, text=f"模型: {model_name}", font=("Arial", 10, "bold")).pack(pady=5)
        ttk.Label(frame, text=f"{price_type} (¥/M tokens):", font=("Arial", 9)).pack(pady=5)
        
        price_entry = ttk.Entry(frame, width=20, font=("Arial", 10))
        price_entry.insert(0, str(current_value))
        price_entry.pack(pady=10)
        price_entry.focus()
        
        def save_price():
            new_value = price_entry.get().strip()
            try:
                float(new_value)  # 验证是数字
                values[col_index] = new_value
                tree.item(row_id, values=values)
                edit_window.destroy()
            except ValueError:
                messagebox.showerror("错误", "请输入有效的数字！", parent=edit_window)
        
        ttk.Button(frame, text="保存", command=save_price).pack(pady=5)
        
        price_entry.bind('<Return>', lambda e: save_price())
    
    def save_all_model_prices(self, tree, window, auto=False):
        """保存所有模型价格到配置"""
        model_prices = {}
        for item in tree.get_children():
            values = tree.item(item)['values']
            model_name = values[0]
            engine = values[1]
            ippm = values[2]
            oppm = values[3]
            cippm = values[4] if len(values) > 4 else '1.0'
            model_prices[model_name] = {
                'engine': str(engine),
                'ippm': str(ippm),
                'oppm': str(oppm),
                'cippm': str(cippm)
            }
        self.config['model_prices'] = model_prices
        self.save_config()
        self.send_prices_to_starfire()
        self.starfire_log(f"✓ 已保存并同步 {len(model_prices)} 个模型的价格配置", "green")
        # 仅非自动保存时弹窗
        if not auto:
            messagebox.showinfo("成功", f"已保存 {len(model_prices)} 个模型的价格配置！", parent=window)
    
    def send_prices_to_starfire(self):
        """通过TCP发送价格配置到starfire.exe"""
        try:
            model_prices = self.config.get('model_prices', {})
            available_models = self.get_all_available_models()
            models_data = []
            
            if model_prices:
                for model_name, prices in model_prices.items():
                    # 使用配置中存储的引擎类型，如果没有则从可用模型中获取
                    engine = str(prices.get('engine') or available_models.get(model_name, 'ollama'))
                    models_data.append({
                        'model': model_name,
                        'engine': engine,
                        'ippm': str(prices.get('ippm', self.config.get('ippm', '3.8'))),
                        'oppm': str(prices.get('oppm', self.config.get('oppm', '8.3'))),
                        'cippm': str(prices.get('cippm', '1.0'))
                    })
            else:
                # 如果没有配置任何模型价格，使用当前模式的引擎作为默认
                model_mode = self.model_mode_var.get()
                engine_map = {
                    'ollama': 'ollama',
                    'proxy': 'openai',
                    'vllm': 'vllm',
                    'llamacpp': 'llama.cpp'
                }
                default_engine = engine_map.get(model_mode, 'ollama')
                self.starfire_log("⚠️ 没有配置模型价格，将发送默认价格配置", "orange")
                models_data.append({
                    'model': '*',
                    'engine': default_engine,
                    'ippm': str(self.config.get('ippm', '3.8')),
                    'oppm': str(self.config.get('oppm', '8.3')),
                    'cippm': '1.0'
                })
            
            message = {
                'id': 'model_price_config',
                'type': 'model_prices',
                'timestamp': int(datetime.now().timestamp()),
                'data': models_data
            }
            message_json = json.dumps(message, ensure_ascii=False)
            self.pending_price_message = message_json
            self.starfire_log(f"📋 准备发送的消息: {message_json[:200]}...", "gray")
            
            tcp_status = False
            sent_count = 0
            if self.tcp_server and hasattr(self.tcp_server, 'clients'):
                with self.tcp_server.clients_lock:
                    client_count = len(self.tcp_server.clients)
                if client_count > 0:
                    sent_count = self.tcp_server.send_to_all_clients(message_json)
                    tcp_status = True
            
            # 统计各引擎的模型数量
            engine_counts = {}
            for model_data in models_data:
                eng = model_data['engine']
                engine_counts[eng] = engine_counts.get(eng, 0) + 1
            
            engine_info = ', '.join([f"{eng}:{cnt}" for eng, cnt in engine_counts.items()])
            
            if tcp_status and sent_count > 0:
                self.starfire_log(f"✓ 价格配置已通过TCP发送到 {sent_count} 个客户端 (模型: {len(models_data)}, 引擎: {engine_info})", "green")
            else:
                self.starfire_log(f"✓ 价格配置已缓存，等待TCP客户端连接 (模型: {len(models_data)}, 引擎: {engine_info})", "blue")
        except Exception as e:
            self.starfire_log(f"❌ 准备价格配置失败: {str(e)}", "red")
            self.starfire_log(f"详细错误: {traceback.format_exc()}", "red")
    
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
        
        is_valid, err_msg = validate_host(host)
        if not is_valid:
            messagebox.showwarning("提示", f"服务器地址格式错误: {err_msg}\n\n正确格式示例:\n• http://111.228.58.164\n• https://chat.example.com\n• http://123.12.1.123:8080")
            return
        
        # 获取注册token
        token = self.get_register_token()
        if not token:
            messagebox.showwarning("配置不完整", "请先登录以获取注册Token！")
            return
        
        model_mode = self.model_mode_var.get()
        
        if not host:
            messagebox.showwarning("配置不完整", "请填写服务器地址！")
            return
        
        # 代理模式需要额外检查配置
        if model_mode == 'proxy':
            proxy_url = self.proxy_base_url_entry.get().strip()
            proxy_key = self.proxy_api_key_entry.get().strip()
            if not all([proxy_url, proxy_key]):
                messagebox.showwarning("配置不完整", "代理模式需要配置 Base URL 和 API Key！")
                return
            print("123 proxy_url:", proxy_url)
            is_valid, err_msg = validate_url(proxy_url)
            if not is_valid:
                messagebox.showwarning("提示", f"Base URL格式错误: {err_msg}\n\n正确格式示例:\n• https://chat.example.com/v1\n• http://123.12.1.123:8080/v1\n• https://chat.example.com/v1/\n• http://123.12.1.123:8080/chat/v1/")
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
            # 基础命令参数（不传递ippm/oppm，价格由配置文件通过TCP发送）
            cmd = [
                starfire_exe,
                "-host", host,
                "-token", token
            ]
            
            # 根据模型模式添加额外参数
            if model_mode == 'proxy':
                proxy_url = self.proxy_base_url_entry.get().strip()
                proxy_key = self.proxy_api_key_entry.get().strip()
                cmd.extend([
                    "-engine", "all",
                    "-openai-url", proxy_url,
                    "-openai-key", proxy_key
                ])
            
            self.starfire_log("=" * 50, "blue")
            self.starfire_log(f"正在启动 Starfire 算力注册...", "blue")
            self.starfire_log(f"模型模式: {model_mode}", "blue")
            self.starfire_log(f"服务器: {host}", "blue")
            if model_mode == 'proxy':
                self.starfire_log(f"代理地址: {proxy_url}", "blue")
            self.starfire_log(f"价格将通过TCP从配置文件同步", "blue")
            
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
            
            # 自动准备价格配置，starfire.exe通过TCP连接后会自动发送
            self.send_prices_to_starfire()
            self.starfire_log("✓ 价格配置已准备，等待starfire.exe连接后自动同步", "blue")
            
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
            
            # 当客户端连接时，如果有待发送的价格配置，立即发送
            if self.pending_price_message and self.tcp_server:
                try:
                    sent_count = self.tcp_server.send_to_all_clients(self.pending_price_message)
                    if sent_count > 0:
                        self.starfire_log(f"📤 已发送价格配置到 {sent_count} 个客户端", "green")
                        # 发送成功后清空待发送消息（可选，根据需求决定是否每次连接都发送）
                        # self.pending_price_message = None
                    else:
                        self.starfire_log(f"⚠️ 没有客户端接收价格配置", "orange")
                except Exception as e:
                    self.starfire_log(f"❌ 发送价格配置失败: {str(e)}", "red")
                    
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
                    
                    # 更新界面显示
                    def _update_income_ui():
                        self.total_income_label.config(text=f"{total:.6f} {currency}")
                        self.latest_income_label.config(text=f"{amount:.6f} {currency}")
                    self.root.after(0, _update_income_ui)
                    
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
                    
                    # 更新界面显示
                    def _update_income_ui():
                        self.total_income_label.config(text=f"{self.total_income:.6f} {currency}")
                        self.latest_income_label.config(text=f"{float(amount):.6f} {currency}")
                    self.root.after(0, _update_income_ui)
                    
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
                    
                    # 更新界面显示
                    def _update_income_ui():
                        self.total_income_label.config(text=f"{self.total_income:.6f} {currency}")
                        self.latest_income_label.config(text=f"{float(amount):.6f} {currency}")
                    self.root.after(0, _update_income_ui)
                    
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
        mode = self.model_mode_var.get()
        if mode == 'proxy':
            total = len(self.model_tree.get_children())
            if total > 0:
                self.running_label.config(text=f"● 代理模型 {total} 个")
            else:
                self.running_label.config(text="")
        else:
            if self.running_models:
                running_list = ", ".join(list(self.running_models)[:2])
                if len(self.running_models) > 2:
                    running_list += f" +{len(self.running_models)-2}"
                self.running_label.config(text=f"● {running_list}")
            else:
                self.running_label.config(text="")
    
    def update_model_colors(self):
        mode = self.model_mode_var.get()
        for item in self.model_tree.get_children():
            values = self.model_tree.item(item)['values']
            if len(values) >= 2:
                model_name = values[1]
                # 代理模式下所有模型视为运行中
                if mode == 'proxy':
                    self.model_tree.item(item, tags=('running',))
                else:
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

            mode = self.model_mode_var.get()
            if mode == 'proxy':
                # 使用代理接口获取模型
                all_models = self.get_all_available_models()
                models = [name for name, engine in all_models.items() if engine == 'openai']
                if not models:
                    self.log("代理模式下未获取到模型", "orange")
                    messagebox.showinfo("提示", "代理模式未获取到模型\n请检查 Base URL 与 API Key")
                    return

                category_count = {}
                for name in models:
                    category = self.get_model_category(name)
                    icon = self.get_category_icon(category)
                    category_name = self.get_category_name(category)
                    category_display = f"{icon} {category_name}"

                    category_count[category] = category_count.get(category, 0) + 1

                    # 代理模式无大小/修改时间信息
                    self.model_tree.insert(
                        "",
                        tk.END,
                        values=(category_display, name, "-", "-")
                    )

                self.update_model_colors()
                self.update_running_label()
                total = len(models)
                category_info = ", ".join([f"{self.get_category_name(cat)}: {count}" for cat, count in category_count.items()])
                self.log(f"成功加载 {total} 个代理模型 ({category_info})", "green")
                # 代理模式不支持本地运行/停止
                self.run_btn.config(state=tk.DISABLED)
                self.stop_btn.config(state=tk.DISABLED)
            else:
                # Ollama 本地列表
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
        """打开Starfire官网（使用配置中的服务器地址）"""
        host = self.host_entry.get().strip()
        
        if not host:
            messagebox.showwarning("提示", "请先填写服务器地址！")
            return
        
        # 动态拼接URL
        if host.startswith('http://') or host.startswith('https://'):
            url = host
        else:
            url = f"http://{host}/"
        
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