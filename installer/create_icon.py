#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
创建一个简单的应用图标
"""

from PIL import Image, ImageDraw, ImageFont
import os

def create_icon():
    """创建一个简单的图标"""
    # 创建 256x256 的图像
    size = 256
    img = Image.new('RGB', (size, size), color='#2C3E50')
    draw = ImageDraw.Draw(img)
    
    # 绘制一个星星和火焰的简单图案
    # 绘制圆形背景
    draw.ellipse([40, 40, 216, 216], fill='#E74C3C', outline='#C0392B', width=5)
    
    # 绘制星星（简化版）
    star_points = [
        (128, 70), (145, 110), (190, 110), (155, 140),
        (170, 185), (128, 155), (86, 185), (101, 140),
        (66, 110), (111, 110)
    ]
    draw.polygon(star_points, fill='#F39C12', outline='#D68910')
    
    # 保存为 PNG
    img.save('icon.png', 'PNG')
    print("✓ 图标已创建: icon.png")
    
    # 转换为 ICO（需要安装 pillow）
    try:
        img.save('icon.ico', format='ICO', sizes=[(256, 256), (128, 128), (64, 64), (48, 48), (32, 32), (16, 16)])
        print("✓ ICO 图标已创建: icon.ico")
    except Exception as e:
        print(f"⚠ 创建 ICO 失败: {e}")
        print("提示: 运行 'pip install pillow' 安装依赖")

if __name__ == "__main__":
    try:
        create_icon()
    except ImportError:
        print("请先安装 Pillow: pip install pillow")