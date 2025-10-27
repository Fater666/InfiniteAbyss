// 全局状态
const state = {
    character: null,
    world: null,
    story: null,
    scene: null,
    charState: null,
    apiConfig: null
};

// API配置管理
const APIConfig = {
    // 从localStorage加载配置
    load() {
        const saved = localStorage.getItem('api_config');
        if (saved) {
            try {
                state.apiConfig = JSON.parse(saved);
                console.log('✅ 已加载API配置');
                return state.apiConfig;
            } catch (e) {
                console.warn('⚠️ API配置解析失败', e);
            }
        }
        return null;
    },

    // 保存配置到localStorage
    save(config) {
        state.apiConfig = config;
        localStorage.setItem('api_config', JSON.stringify(config));
        console.log('💾 API配置已保存');
    },

    // 清除配置
    clear() {
        state.apiConfig = null;
        localStorage.removeItem('api_config');
        console.log('🗑️ API配置已清除');
    },

    // 获取当前配置
    get() {
        return state.apiConfig;
    },

    // 获取请求头（用于API调用）
    getHeaders() {
        const config = this.get();
        const headers = {
            'Content-Type': 'application/json'
        };

        // 如果有自定义API配置，添加到headers
        if (config && config.apiKey) {
            headers['X-Custom-API-Key'] = config.apiKey;
            headers['X-Custom-API-Base'] = config.apiBaseUrl;
            headers['X-Custom-API-Model'] = config.model;
            console.log('🔧 使用自定义API配置:', {
                baseUrl: config.apiBaseUrl,
                model: config.model,
                apiKey: config.apiKey.substring(0, 10) + '...'
            });
        } else {
            console.log('ℹ️ 使用服务器默认API配置');
        }

        return headers;
    }
};

// 页面加载时初始化API配置
APIConfig.load();

// API 调用
const API = {
    async createCharacter(charData) {
        const res = await fetch('/api/characters', {
            method: 'POST',
            headers: APIConfig.getHeaders(),
            body: JSON.stringify(charData)
        });
        const data = await res.json();
        if (!res.ok) {
            throw new Error(data.error || '创建失败');
        }
        return data;
    },

    async generateCharacter(name, gender, age, prompt) {
        const res = await fetch('/api/characters/generate', {
            method: 'POST',
            headers: APIConfig.getHeaders(),
            body: JSON.stringify({ name, gender, age, prompt })
        });
        const data = await res.json();
        if (!res.ok) {
            throw new Error(data.error || '生成失败');
        }
        return data;
    },

    async listCharacters() {
        const res = await fetch('/api/characters', {
            headers: APIConfig.getHeaders()
        });
        const data = await res.json();
        if (!res.ok) {
            throw new Error(data.error || '获取角色列表失败');
        }
        return data;
    },

    async parseSegment(segmentText) {
        const res = await fetch('/api/worlds/parse', {
            method: 'POST',
            headers: APIConfig.getHeaders(),
            body: JSON.stringify({ segment_text: segmentText })
        });
        return res.json();
    },

    async startStory(characterID, worldID) {
        const res = await fetch('/api/stories/start', {
            method: 'POST',
            headers: APIConfig.getHeaders(),
            body: JSON.stringify({ character_id: characterID, world_id: worldID })
        });
        return res.json();
    },

    async takeAction(storyID, action) {
        const res = await fetch('/api/stories/action', {
            method: 'POST',
            headers: APIConfig.getHeaders(),
            body: JSON.stringify({ story_id: storyID, action })
        });
        return res.json();
    },

    async undoTurn(storyID) {
        const res = await fetch('/api/stories/undo', {
            method: 'POST',
            headers: APIConfig.getHeaders(),
            body: JSON.stringify({ story_id: storyID })
        });
        return res.json();
    },

    async saveGame(storyID, name, description) {
        const res = await fetch('/api/saves', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ story_id: storyID, name, description })
        });
        return res.json();
    },

    async listSaves(characterID) {
        const res = await fetch(`/api/saves?character_id=${characterID}`);
        return res.json();
    },

    async loadGame(storyID) {
        const res = await fetch('/api/saves/load', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ story_id: storyID })
        });
        return res.json();
    }
};

// UI 更新函数
const UI = {
    showCharacterInfo(character) {
        const info = document.getElementById('character-info');
        const genderIcon = character.gender === 'female' ? '♀️' : '♂️';
        info.innerHTML = `
            <h3>${genderIcon} ${character.name}</h3>
            <p>年龄: ${character.age} | 等级: ${character.level}</p>
            <p>经验: ${character.xp} XP</p>
            ${character.background ? `<details style="margin-top: 10px; font-size: 0.9em;">
                <summary style="cursor: pointer; font-weight: bold;">查看人设</summary>
                <div style="padding: 10px 0;">
                    <p><strong>外貌：</strong>${character.appearance || '未设定'}</p>
                    <p><strong>性格：</strong>${character.personality || '未设定'}</p>
                    <p><strong>背景：</strong>${character.background || '未设定'}</p>
                </div>
            </details>` : ''}
            <p class="hint">准备进入无限流世界...</p>
        `;
    },

    showCharacterState(charState) {
        const statePanel = document.getElementById('character-state');
        statePanel.style.display = 'block';

        // 检查charState是否有必要的字段
        if (!charState || typeof charState.hp === 'undefined') {
            console.error('⚠️ charState数据不完整:', charState);
            document.getElementById('hp-text').textContent = 'N/A';
            document.getElementById('san-text').textContent = 'N/A';
            return;
        }

        // 更新HP
        const hpPercent = (charState.hp / charState.max_hp) * 100;
        document.getElementById('hp-bar').style.width = `${hpPercent}%`;
        document.getElementById('hp-text').textContent = `${charState.hp}/${charState.max_hp}`;

        // 更新理智
        const sanPercent = (charState.san / charState.max_san) * 100;
        document.getElementById('san-bar').style.width = `${sanPercent}%`;
        document.getElementById('san-text').textContent = `${charState.san}/${charState.max_san}`;

        // 更新属性
        const attrsDiv = document.getElementById('attributes');
        const attributes = charState.attributes || {};
        if (Object.keys(attributes).length > 0) {
            attrsDiv.innerHTML = Object.entries(attributes)
                .map(([name, value]) => `
                    <div class="attribute">
                        <div class="attribute-name">${this.translateAttr(name)}</div>
                        <div class="attribute-value">${value}</div>
                    </div>
                `).join('');
        } else {
            attrsDiv.innerHTML = '<div style="color: #888;">暂无属性数据</div>';
        }
    },

    translateAttr(attr) {
        const map = {
            strength: '力量',
            dexterity: '敏捷',
            intelligence: '智力',
            charisma: '魅力',
            perception: '感知'
        };
        return map[attr] || attr;
    },

    showWorldInfo(world) {
        const worldInfo = document.getElementById('world-info');
        worldInfo.style.display = 'block';

        document.getElementById('world-name').textContent = world.name || '未知世界';
        document.getElementById('world-description').textContent = world.description || '暂无描述';
        document.getElementById('world-genre').textContent = this.translateGenre(world.genre);
        document.getElementById('world-difficulty').textContent = `难度: ${'★'.repeat(world.difficulty || 5)}`;

        // 安全地显示目标（确保 goals 是数组）
        const goalsDiv = document.getElementById('world-goals');
        const goals = Array.isArray(world.goals) ? world.goals : [];
        if (goals.length > 0) {
            goalsDiv.innerHTML = '<h3>通关目标</h3>' + goals.map(goal =>
                `<div class="goal-item">✓ ${goal}</div>`
            ).join('');
        } else {
            goalsDiv.innerHTML = '<h3>通关目标</h3><div class="goal-item">自由探索</div>';
        }

        // 安全地显示NPC（确保 npcs 是数组）
        const npcList = document.getElementById('npc-list');
        const npcs = Array.isArray(world.npcs) ? world.npcs : [];
        if (npcs.length > 0) {
            npcList.innerHTML = npcs.map(npc => `
                <div class="npc-item">
                    <div class="npc-name">${npc.name || '未知'}</div>
                    <div class="npc-role">${this.translateRole(npc.role)}</div>
                    <div style="font-size: 0.85em; color: #a8a8a8; margin-top: 5px;">${npc.description || ''}</div>
                </div>
            `).join('');
        } else {
            npcList.innerHTML = '<div style="color: #888; padding: 10px;">暂无关键角色</div>';
        }

        // 安全地显示目标列表
        const goalsList = document.getElementById('goals-list');
        if (goals.length > 0) {
            goalsList.innerHTML = goals.map(goal => `<li>${goal}</li>`).join('');
        } else {
            goalsList.innerHTML = '<li>自由探索这个世界</li>';
        }
    },

    translateGenre(genre) {
        const map = {
            romance: '💕 浪漫',
            adult: '🔞 成人',
            harem: '👥 后宫',
            fantasy: '⚔️ 奇幻',
            urban: '🏙️ 都市',
            scifi: '🚀 科幻',
            horror: '🌙 暗黑',
            mystery: '🔍 悬疑'
        };
        return map[genre] || genre;
    },

    translateRole(role) {
        const map = {
            love_interest: '💖 恋爱对象',
            rival: '⚔️ 竞争对手',
            mentor: '👤 导师',
            target: '🎯 目标',
            ally: '🤝 盟友',
            enemy: '⚠️ 敌人',
            neutral: '😐 中立',
            boss: '👑 首领',
            friend: '👥 朋友',
            potential_companion: '💫 潜在伙伴'
        };
        return map[role] || role;
    },

    showNarrative(story) {
        const logDiv = document.getElementById('narrative-log');
        logDiv.style.display = 'block';

        const logContent = document.getElementById('log-content');
        const narrative = Array.isArray(story.narrative) ? story.narrative : [];
        logContent.innerHTML = narrative.map(entry => {
            let diceInfo = '';
            if (entry.dice_roll) {
                const dr = entry.dice_roll;
                const successClass = dr.success ? 'success' : '';
                const criticalClass = dr.critical ? 'critical' : '';
                diceInfo = `<div class="dice-roll ${successClass} ${criticalClass}">
                    🎲 ${dr.result} + ${dr.modifier} = ${dr.result + dr.modifier} 
                    (目标: ${dr.target}) 
                    ${dr.critical ? (dr.success ? '大成功!' : '大失败!') : (dr.success ? '成功' : '失败')}
                </div>`;
            }
            return `
                <div class="log-entry ${entry.type}">
                    <div style="opacity: 0.7; font-size: 0.9em; margin-bottom: 5px;">
                        回合 ${entry.turn} · ${this.translateType(entry.type)}
                    </div>
                    ${entry.content}
                    ${diceInfo}
                </div>
            `;
        }).join('');

        // 滚动到底部
        logContent.scrollTop = logContent.scrollHeight;
    },

    translateType(type) {
        const map = {
            system: '系统',
            action: '行动',
            result: '结果',
            dialogue: '对话'
        };
        return map[type] || type;
    },

    showOptions(options) {
        const optionsDiv = document.getElementById('action-options');
        optionsDiv.style.display = 'block';

        const optionsList = document.getElementById('options-list');
        const validOptions = Array.isArray(options) ? options : [];
        optionsList.innerHTML = validOptions.map(opt => `
            <button class="option-btn" data-option='${JSON.stringify(opt)}'>
                <span class="option-label">${opt.label}</span>
                <div class="option-description">${opt.description}</div>
                <div class="option-meta">
                    难度: ${opt.difficulty} | 
                    风险: <span class="risk-${opt.risk}">${opt.risk === 'low' ? '低' : opt.risk === 'medium' ? '中' : '高'}</span>
                </div>
            </button>
        `).join('');

        // 绑定选项点击事件  
        document.querySelectorAll('.option-btn').forEach(btn => {
            btn.onclick = () => {
                const opt = JSON.parse(btn.dataset.option);
                // 弹出输入框让用户输入具体行动
                const detail = prompt(`请输入具体行动内容（默认：${opt.label}）`, opt.description);
                if (detail !== null) {  // null表示用户取消
                    this.executeAction({
                        type: opt.action_type,
                        content: detail || opt.label  // 如果为空，使用label
                    });
                }
            };
        });
    },

    async executeAction(action) {
        if (!state.story) return;

        // 禁用所有按钮
        document.querySelectorAll('.option-btn, #custom-action-btn').forEach(btn => {
            btn.disabled = true;
            btn.style.opacity = '0.5';
        });

        try {
            const result = await API.takeAction(state.story.id, action);

            // 更新状态
            state.story = result.story;

            // 更新UI
            this.showNarrative(state.story);

            if (result.result.scene_end) {
                // 场景结束
                const logContent = document.getElementById('log-content');
                logContent.innerHTML += `
                    <div class="log-entry system">
                        <h3>🎯 场景结束</h3>
                        <p>${state.story.status === 'completed' ? '你成功通过了这个世界！' : '你在这个世界失败了...'}</p>
                        <button class="btn btn-primary" onclick="location.reload()">进入下一个世界</button>
                    </div>
                `;
                document.getElementById('action-options').style.display = 'none';
            } else {
                // 显示新选项
                this.showOptions(result.result.next_options);
            }

            // 更新角色状态
            // 需要重新获取
            if (state.character && state.world) {
                // 简化：从changes中更新
                if (state.charState) {
                    state.charState.hp += result.result.changes.hp_change || 0;
                    state.charState.san += result.result.changes.san_change || 0;
                    this.showCharacterState(state.charState);
                }
            }

        } catch (error) {
            alert('执行行动失败: ' + error.message);
        } finally {
            // 重新启用按钮
            document.querySelectorAll('.option-btn, #custom-action-btn').forEach(btn => {
                btn.disabled = false;
                btn.style.opacity = '1';
            });
        }
    },

    hideSegmentInput() {
        document.getElementById('segment-input-section').style.display = 'none';
    },

    async undoLastTurn() {
        if (!state.story) return;

        if (!confirm('确定要回退到上一回合吗？')) return;

        try {
            const result = await API.undoTurn(state.story.id);
            state.story = result.story;
            state.charState = result.char_state;

            this.showNarrative(state.story);
            this.showCharacterState(state.charState);

            alert('✅ 已回退到上一回合');
        } catch (error) {
            alert('回退失败: ' + error.message);
        }
    },

    async saveCurrentGame() {
        if (!state.story) return;

        const name = prompt('请输入存档名称：', `存档 - 回合${state.story.turn}`);
        if (!name) return;

        try {
            await API.saveGame(state.story.id, name, '');
            alert('💾 存档成功！');
        } catch (error) {
            alert('存档失败: ' + error.message);
        }
    },

    async showLoadMenu() {
        if (!state.character) {
            alert('请先创建角色');
            return;
        }

        try {
            const result = await API.listSaves(state.character.id);
            const saves = result.saves || [];

            if (saves.length === 0) {
                alert('暂无存档');
                return;
            }

            const saveList = saves.map((save, index) =>
                `${index + 1}. ${save.name} (${save.description || '回合' + save.turn})`
            ).join('\n');

            const choice = prompt(`选择要读取的存档（输入序号）：\n\n${saveList}`);
            if (!choice) return;

            const index = parseInt(choice) - 1;
            if (index >= 0 && index < saves.length) {
                await this.loadSaveGame(saves[index].story_id);
            }
        } catch (error) {
            alert('读档失败: ' + error.message);
        }
    },

    async loadSaveGame(storyID) {
        try {
            const result = await API.loadGame(storyID);
            state.story = result.story;
            state.scene = result.scene;
            state.charState = result.char_state;

            // 更新UI
            this.showNarrative(state.story);
            this.showCharacterState(state.charState);
            document.getElementById('narrative-log').style.display = 'block';
            document.getElementById('action-options').style.display = 'block';

            alert('📂 读档成功！');
        } catch (error) {
            alert('读档失败: ' + error.message);
        }
    }
};

// 初始化
document.addEventListener('DOMContentLoaded', () => {
    // 创建角色按钮
    document.getElementById('create-character-btn').onclick = () => {
        document.getElementById('create-character-modal').classList.add('show');
    };

    // 加载角色
    document.getElementById('load-character-btn').onclick = async () => {
        try {
            const characters = await API.listCharacters();

            if (!characters || characters.length === 0) {
                alert('还没有保存的角色，请先创建角色');
                return;
            }

            const charactersList = document.getElementById('characters-list');
            charactersList.innerHTML = characters.map(char => `
                <div class="character-item" style="border: 1px solid #444; padding: 15px; margin-bottom: 10px; border-radius: 5px; cursor: pointer; transition: all 0.3s;"
                     onmouseover="this.style.background='#2a2a2a'" onmouseout="this.style.background='transparent'"
                     onclick="loadCharacterById('${char.id}')">
                    <div style="display: flex; justify-content: space-between; align-items: center;">
                        <div>
                            <h3 style="margin: 0 0 5px 0;">${char.name}</h3>
                            <p style="margin: 0; color: #888; font-size: 0.9em;">
                                ${char.gender === 'male' ? '♂️ 男' : '♀️ 女'} | ${char.age}岁 | Lv.${char.level}
                            </p>
                        </div>
                        <div style="text-align: right; color: #888; font-size: 0.85em;">
                            创建于: ${new Date(char.created_at).toLocaleDateString()}
                        </div>
                    </div>
                    ${char.appearance ? `<p style="margin: 10px 0 0 0; color: #aaa; font-size: 0.9em;">${char.appearance.substring(0, 80)}...</p>` : ''}
                </div>
            `).join('');

            document.getElementById('load-character-modal').classList.add('show');
        } catch (error) {
            console.error('加载角色列表失败:', error);
            alert('加载角色列表失败: ' + error.message);
        }
    };

    // 全局函数：根据ID加载角色
    window.loadCharacterById = async (characterId) => {
        try {
            const character = await fetch(`/api/characters/${characterId}`).then(res => res.json());

            if (character.error) {
                throw new Error(character.error);
            }

            state.character = character;
            UI.showCharacterInfo(character);
            document.getElementById('load-character-modal').classList.remove('show');

            // 显示段落输入
            document.getElementById('segment-input-section').style.display = 'block';

            alert(`✅ 成功加载角色：${character.name}`);
        } catch (error) {
            console.error('加载角色失败:', error);
            alert('加载角色失败: ' + error.message);
        }
    };

    // 取消加载角色
    document.getElementById('cancel-load-character').onclick = () => {
        document.getElementById('load-character-modal').classList.remove('show');
    };

    // 取消创建角色
    document.getElementById('cancel-create-character').onclick = () => {
        document.getElementById('create-character-modal').classList.remove('show');
    };

    // 创建方式切换
    document.getElementById('creation-mode').onchange = (e) => {
        const mode = e.target.value;
        if (mode === 'ai') {
            document.getElementById('ai-mode-section').style.display = 'block';
            document.getElementById('manual-mode-section').style.display = 'none';
        } else {
            document.getElementById('ai-mode-section').style.display = 'none';
            document.getElementById('manual-mode-section').style.display = 'block';
        }
    };

    // 手动模式属性计算
    const attrInputs = ['attr-strength', 'attr-dexterity', 'attr-intelligence', 'attr-charisma', 'attr-perception'];
    attrInputs.forEach(id => {
        document.getElementById(id).oninput = () => {
            let total = 0;
            attrInputs.forEach(attrId => {
                total += parseInt(document.getElementById(attrId).value) || 0;
            });
            document.getElementById('attr-total').textContent = total;
        };
    });

    // 确认创建角色
    document.getElementById('confirm-create-character').onclick = async () => {
        const name = document.getElementById('character-name').value.trim();
        const gender = document.getElementById('character-gender').value;
        const age = parseInt(document.getElementById('character-age').value);
        const mode = document.getElementById('creation-mode').value;

        if (!name) {
            alert('请输入角色名字');
            return;
        }

        const btn = document.getElementById('confirm-create-character');
        btn.disabled = true;
        btn.textContent = mode === 'ai' ? 'AI生成中...' : '创建中...';

        try {
            let character;

            if (mode === 'ai') {
                // AI自动生成
                const prompt = document.getElementById('character-prompt').value.trim();
                const result = await API.generateCharacter(name, gender, age, prompt);

                // 检查是否有错误
                if (result.error) {
                    throw new Error(result.error);
                }

                character = result;

                // 验证返回的数据
                if (!character.appearance || !character.personality || !character.background) {
                    throw new Error('AI生成的数据不完整，请重试');
                }

                alert(`✨ AI生成成功！\n\n外貌：${character.appearance.substring(0, 50)}...\n\n点击角色卡"查看人设"可查看完整信息`);
            } else {
                // 手动创建
                const charData = {
                    name,
                    gender,
                    age,
                    appearance: document.getElementById('character-appearance').value.trim(),
                    personality: document.getElementById('character-personality').value.trim(),
                    background: document.getElementById('character-background').value.trim(),
                    base_attributes: {
                        strength: parseInt(document.getElementById('attr-strength').value),
                        dexterity: parseInt(document.getElementById('attr-dexterity').value),
                        intelligence: parseInt(document.getElementById('attr-intelligence').value),
                        charisma: parseInt(document.getElementById('attr-charisma').value),
                        perception: parseInt(document.getElementById('attr-perception').value)
                    }
                };
                const result = await API.createCharacter(charData);

                if (result.error) {
                    throw new Error(result.error);
                }

                character = result;
            }

            state.character = character;
            UI.showCharacterInfo(character);
            document.getElementById('create-character-modal').classList.remove('show');

            // 显示段落输入
            document.getElementById('segment-input-section').style.display = 'block';
        } catch (error) {
            console.error('创建角色错误:', error);
            alert('创建角色失败: ' + (error.message || error.error || '未知错误'));
        } finally {
            btn.disabled = false;
            btn.textContent = '创建角色';
        }
    };

    // 解析段落
    document.getElementById('parse-segment-btn').onclick = async () => {
        const segmentText = document.getElementById('segment-text').value.trim();
        if (!segmentText) {
            alert('请输入小说段落');
            return;
        }

        const btn = document.getElementById('parse-segment-btn');
        btn.disabled = true;
        btn.textContent = '正在解析...';

        try {
            const world = await API.parseSegment(segmentText);

            // 检查返回的数据是否有效
            if (!world || world.error) {
                throw new Error(world?.error || '解析返回数据无效');
            }

            // 确保必要的字段存在
            if (!world.id || !world.name) {
                throw new Error('世界数据不完整，请重试');
            }

            // 确保数组字段存在
            world.goals = world.goals || [];
            world.npcs = world.npcs || [];
            world.plot_lines = world.plot_lines || [];

            state.world = world;
            UI.showWorldInfo(world);
            UI.hideSegmentInput();
        } catch (error) {
            console.error('解析错误:', error);
            alert('解析失败: ' + (error.message || error.error || '未知错误'));
        } finally {
            btn.disabled = false;
            btn.textContent = '生成世界';
        }
    };

    // 开始冒险
    document.getElementById('start-adventure-btn').onclick = async () => {
        if (!state.character || !state.world) {
            alert('请先创建角色和世界');
            return;
        }

        const btn = document.getElementById('start-adventure-btn');
        btn.disabled = true;
        btn.textContent = '正在进入...';

        try {
            const result = await API.startStory(state.character.id, state.world.id);
            console.log('📦 API返回的数据:', result);

            state.story = result.story;
            state.scene = result.scene;
            state.charState = result.char_state;

            // 隐藏世界信息，显示故事
            document.getElementById('world-info').style.display = 'none';

            // 确保char_state存在
            if (!state.charState) {
                console.error('⚠️ 后端没有返回char_state');
                alert('开始冒险失败：后端返回数据不完整');
                return;
            }

            UI.showCharacterState(state.charState);
            UI.showNarrative(state.story);

            // 生成初始选项（需要调用一次）
            // 暂时使用默认选项
            UI.showOptions([
                {
                    id: 'opt_1',
                    label: '观察四周',
                    description: '仔细观察周围的环境，寻找线索',
                    action_type: 'investigate',
                    difficulty: 10,
                    risk: 'low'
                },
                {
                    id: 'opt_2',
                    label: '向前探索',
                    description: '小心地向前移动，探索未知区域',
                    action_type: 'move',
                    difficulty: 12,
                    risk: 'medium'
                },
                {
                    id: 'opt_3',
                    label: '保持警惕',
                    description: '站在原地，观察周围的动静',
                    action_type: 'custom',
                    difficulty: 8,
                    risk: 'low'
                }
            ]);

        } catch (error) {
            alert('开始冒险失败: ' + error.message);
            btn.disabled = false;
            btn.textContent = '开始冒险';
        }
    };

    // 自定义行动
    document.getElementById('custom-action-btn').onclick = () => {
        const input = document.getElementById('custom-action-input');
        const content = input.value.trim();
        if (!content) {
            alert('请输入行动内容');
            return;
        }

        UI.executeAction({
            type: 'custom',
            content: content
        });

        input.value = '';
    };

    // 回车执行自定义行动
    document.getElementById('custom-action-input').onkeypress = (e) => {
        if (e.key === 'Enter') {
            document.getElementById('custom-action-btn').click();
        }
    };

    // ========== API设置相关 ==========

    // 显示/隐藏API密钥
    document.getElementById('show-api-key').onchange = (e) => {
        const keyInput = document.getElementById('api-key');
        keyInput.type = e.target.checked ? 'text' : 'password';
    };

    // 打开API设置模态框
    UI.showAPISettings = () => {
        const modal = document.getElementById('api-settings-modal');
        modal.style.display = 'flex';

        // 加载已保存的配置
        const config = APIConfig.get();
        if (config) {
            document.getElementById('api-provider').value = config.provider || 'grok';
            document.getElementById('api-base-url').value = config.apiBaseUrl || 'https://api.x.ai/v1';
            document.getElementById('api-key').value = config.apiKey || '';
            document.getElementById('api-model').value = config.model || 'grok-3';
        } else {
            // 默认值
            document.getElementById('api-provider').value = 'grok';
            document.getElementById('api-base-url').value = 'https://api.x.ai/v1';
            document.getElementById('api-key').value = '';
            document.getElementById('api-model').value = 'grok-3';
        }

        // 隐藏测试结果
        document.getElementById('api-test-result').style.display = 'none';
    };

    // 关闭API设置模态框
    document.getElementById('cancel-api-settings').onclick = () => {
        document.getElementById('api-settings-modal').style.display = 'none';
    };

    // 保存API设置
    document.getElementById('save-api-settings').onclick = () => {
        const provider = document.getElementById('api-provider').value;
        const apiBaseUrl = document.getElementById('api-base-url').value.trim();
        const apiKey = document.getElementById('api-key').value.trim();
        const model = document.getElementById('api-model').value.trim();

        if (!apiBaseUrl) {
            alert('请输入API Base URL');
            return;
        }

        if (!apiKey) {
            alert('请输入API Key');
            return;
        }

        if (!model) {
            alert('请输入模型名称');
            return;
        }

        const config = {
            provider,
            apiBaseUrl,
            apiKey,
            model
        };

        APIConfig.save(config);

        // 显示成功提示
        const resultDiv = document.getElementById('api-test-result');
        resultDiv.style.display = 'block';
        resultDiv.style.background = '#d4edda';
        resultDiv.style.color = '#155724';
        resultDiv.style.border = '1px solid #c3e6cb';
        resultDiv.innerHTML = '✅ API设置已保存！下次调用将使用新配置。';

        setTimeout(() => {
            document.getElementById('api-settings-modal').style.display = 'none';
        }, 1500);
    };

    // 测试API连接
    document.getElementById('test-api-connection').onclick = async () => {
        const apiBaseUrl = document.getElementById('api-base-url').value.trim();
        const apiKey = document.getElementById('api-key').value.trim();
        const model = document.getElementById('api-model').value.trim();

        if (!apiBaseUrl || !apiKey || !model) {
            alert('请先填写完整的API配置');
            return;
        }

        const btn = document.getElementById('test-api-connection');
        const originalText = btn.textContent;
        btn.disabled = true;
        btn.textContent = '🔌 测试中...';

        const resultDiv = document.getElementById('api-test-result');
        resultDiv.style.display = 'block';
        resultDiv.style.background = '#d1ecf1';
        resultDiv.style.color = '#0c5460';
        resultDiv.style.border = '1px solid #bee5eb';
        resultDiv.innerHTML = '⏳ 正在测试连接...';

        try {
            // 测试API连接 - 发送一个简单的测试请求
            const response = await fetch(apiBaseUrl + '/chat/completions', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${apiKey}`
                },
                body: JSON.stringify({
                    model: model,
                    messages: [{ role: 'user', content: 'Hi' }],
                    max_tokens: 10
                })
            });

            if (response.ok) {
                resultDiv.style.background = '#d4edda';
                resultDiv.style.color = '#155724';
                resultDiv.style.border = '1px solid #c3e6cb';
                resultDiv.innerHTML = '✅ API连接测试成功！可以正常使用。';
            } else {
                const error = await response.json();
                throw new Error(error.error?.message || `HTTP ${response.status}`);
            }
        } catch (error) {
            resultDiv.style.background = '#f8d7da';
            resultDiv.style.color = '#721c24';
            resultDiv.style.border = '1px solid #f5c6cb';
            resultDiv.innerHTML = `❌ 连接测试失败：${error.message}<br><small>请检查API Base URL和API Key是否正确</small>`;
        } finally {
            btn.disabled = false;
            btn.textContent = originalText;
        }
    };

    // 清除API设置
    document.getElementById('clear-api-settings').onclick = () => {
        if (confirm('确定要清除API设置吗？将恢复使用服务器默认配置。')) {
            APIConfig.clear();

            // 清空表单
            document.getElementById('api-base-url').value = 'https://api.x.ai/v1';
            document.getElementById('api-key').value = '';
            document.getElementById('api-model').value = 'grok-3';

            const resultDiv = document.getElementById('api-test-result');
            resultDiv.style.display = 'block';
            resultDiv.style.background = '#fff3cd';
            resultDiv.style.color = '#856404';
            resultDiv.style.border = '1px solid #ffeaa7';
            resultDiv.innerHTML = '🗑️ API设置已清除，将使用服务器配置。';

            setTimeout(() => {
                document.getElementById('api-settings-modal').style.display = 'none';
            }, 1500);
        }
    };

    // 点击模态框外部关闭
    window.onclick = (e) => {
        const modal = document.getElementById('api-settings-modal');
        if (e.target === modal) {
            modal.style.display = 'none';
        }
    };

    // 页面加载完成后检查API配置
    const config = APIConfig.get();
    if (config && config.apiKey) {
        console.log('✅ 检测到自定义API配置，将优先使用');
        console.log('📍 API Base:', config.apiBaseUrl);
        console.log('🤖 Model:', config.model);
    } else {
        console.log('ℹ️ 未配置自定义API，将使用服务器config.yml配置');
    }
});

