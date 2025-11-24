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


class OllamaManager:
    def __init__(self, root):
        self.root = root
        self.root.title("StarFire MaaS 算力分享APP")
        self.root.geometry("1000x700")
        self.root.resizable(True, True)
        
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
            'oppm': '8.3'
        }
        
        try:
            if os.path.exists(self.config_file):
                with open(self.config_file, 'r', encoding='utf-8') as f:
                    saved_config = json.load(f)
                    self.config.update(saved_config)
        except:
            pass
    
    def save_config(self):
        try:
            with open(self.config_file, 'w', encoding='utf-8') as f:
                json.dump(self.config, f, indent=2, ensure_ascii=False)
        except Exception as e:
            self.log(f"保存配置失败: {str(e)}", "red")
    
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
        
        self.status_label = ttk.Label(
            top_frame, 
            text="正在检查 Ollama 安装状态...", 
            font=("Arial", 10)
        )
        self.status_label.pack(anchor=tk.W)
        
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
        
        token_frame = ttk.Frame(config_frame)
        token_frame.pack(fill=tk.X, pady=5)
        ttk.Label(token_frame, text="Token:", width=12).pack(side=tk.LEFT)
        self.token_entry = ttk.Entry(token_frame, show="*")
        self.token_entry.insert(0, self.config['token'])
        self.token_entry.pack(side=tk.LEFT, fill=tk.X, expand=True, padx=(5, 0))
        
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
        ttk.Label(ippm_frame, text="¥/M tokens").pack(side=tk.LEFT, padx=(5, 0))
        
        oppm_frame = ttk.Frame(config_frame)
        oppm_frame.pack(fill=tk.X, pady=5)
        ttk.Label(oppm_frame, text="输出价格:", width=12).pack(side=tk.LEFT)
        self.oppm_entry = ttk.Entry(oppm_frame, width=15)
        self.oppm_entry.insert(0, self.config['oppm'])
        self.oppm_entry.pack(side=tk.LEFT, padx=(5, 0))
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
        
        control_buttons = ttk.Frame(control_frame)
        control_buttons.pack(fill=tk.X)
        
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
        
        self.save_config()
        self.starfire_log("✓ 配置已保存", "green")
        messagebox.showinfo("成功", "配置已保存！")
    
    def start_starfire(self):
        host = self.host_entry.get().strip()
        token = self.token_entry.get().strip()
        ippm = self.ippm_entry.get().strip()
        oppm = self.oppm_entry.get().strip()
        
        if not all([host, token, ippm, oppm]):
            messagebox.showwarning("配置不完整", "请填写所有必填配置项！")
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
            cmd = [
                starfire_exe,
                "-host", host,
                "-token", token,
                "-ippm", ippm,
                "-oppm", oppm
            ]
            
            self.starfire_log("=" * 50, "blue")
            self.starfire_log(f"正在启动 Starfire 算力注册...", "blue")
            self.starfire_log(f"服务器: {host}", "blue")
            self.starfire_log(f"输入价格: {ippm} ¥/M tokens", "blue")
            self.starfire_log(f"输出价格: {oppm} ¥/M tokens", "blue")
            self.starfire_log("=" * 50, "blue")
            
            if platform.system() == "Windows":
                self.starfire_process = subprocess.Popen(
                    cmd,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.STDOUT,
                    bufsize=0,
                    creationflags=SUBPROCESS_FLAGS
                )
            else:
                self.starfire_process = subprocess.Popen(
                    cmd,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.STDOUT,
                    text=True,
                    bufsize=1,
                    universal_newlines=True
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
                            if any(keyword in line.lower() for keyword in ['error', 'failed', '失败', '错误']):
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
                        
                        if any(keyword in line.lower() for keyword in ['error', 'failed', '失败', '错误']):
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
                    ["ollama", "run", "--keepalive", "-1m", model_name],
                    stdin=subprocess.PIPE,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.PIPE,
                    text=True,
                    creationflags=SUBPROCESS_FLAGS  # ← 关键修复
                )
            else:
                process = subprocess.Popen(
                    ["ollama", "run", "--keepalive", "-1m", model_name],
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
    app = OllamaManager(root)
    root.mainloop()


if __name__ == "__main__":
    main()