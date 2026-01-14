'use client'

import React, { useState, useCallback, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { 
  Play, Send, Loader2, Copy, Check, ChevronDown,
  Zap, Clock, Coins, AlertCircle, Sparkles, Settings,
  RefreshCw, Key, Globe, Server, CheckCircle, XCircle
} from 'lucide-react'

interface Message {
  role: 'system' | 'user' | 'assistant'
  content: string
}

interface PlaygroundResponse {
  success: boolean
  status_code: number
  latency_ms: number
  response?: any
  error?: string
  input_tokens?: number
  output_tokens?: number
  model?: string
}

interface Model {
  id: string
  object?: string
  created?: number
  owned_by?: string
}

interface APIConfig {
  baseUrl: string
  apiKey: string
}

const DEFAULT_BASE_URL = 'http://127.0.0.1:8317/v1'

export function APIPlayground() {
  // API Configuration
  const [config, setConfig] = useState<APIConfig>({
    baseUrl: DEFAULT_BASE_URL,
    apiKey: '',
  })
  const [showConfig, setShowConfig] = useState(false)
  const [configSaved, setConfigSaved] = useState(false)

  // Models
  const [models, setModels] = useState<Model[]>([])
  const [selectedModel, setSelectedModel] = useState<Model | null>(null)
  const [loadingModels, setLoadingModels] = useState(false)
  const [modelsError, setModelsError] = useState<string | null>(null)
  const [showModelSelector, setShowModelSelector] = useState(false)

  // Messages & Response
  const [messages, setMessages] = useState<Message[]>([
    { role: 'user', content: '' }
  ])
  const [response, setResponse] = useState<PlaygroundResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [copied, setCopied] = useState(false)
  const [streaming, setStreaming] = useState(false)
  const [streamContent, setStreamContent] = useState('')

  // Load config from localStorage on mount
  useEffect(() => {
    const savedConfig = localStorage.getItem('shinapi_playground_config')
    if (savedConfig) {
      try {
        const parsed = JSON.parse(savedConfig)
        setConfig({
          baseUrl: parsed.baseUrl || DEFAULT_BASE_URL,
          apiKey: parsed.apiKey || '',
        })
      } catch {}
    }
  }, [])

  // Fetch models from API
  const fetchModels = useCallback(async () => {
    setLoadingModels(true)
    setModelsError(null)

    try {
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
      }
      if (config.apiKey) {
        headers['Authorization'] = `Bearer ${config.apiKey}`
      }

      const res = await fetch(`${config.baseUrl}/models`, { headers })
      
      if (!res.ok) {
        throw new Error(`Failed to fetch models: ${res.status} ${res.statusText}`)
      }

      const data = await res.json()
      const modelList = data.data || data.models || []
      
      setModels(modelList)
      
      // Auto-select first model if none selected
      if (modelList.length > 0 && !selectedModel) {
        setSelectedModel(modelList[0])
      }
    } catch (err: any) {
      setModelsError(err.message || 'Failed to fetch models')
      // Fallback models
      setModels([
        { id: 'gpt-4o', owned_by: 'openai' },
        { id: 'gpt-4o-mini', owned_by: 'openai' },
        { id: 'claude-sonnet-4-20250514', owned_by: 'anthropic' },
        { id: 'gemini-2.0-flash', owned_by: 'google' },
      ])
    } finally {
      setLoadingModels(false)
    }
  }, [config.baseUrl, config.apiKey, selectedModel])

  // Fetch models on mount and when config changes
  useEffect(() => {
    fetchModels()
  }, []) // Only on mount

  // Save config
  const saveConfig = useCallback(() => {
    localStorage.setItem('shinapi_playground_config', JSON.stringify(config))
    setConfigSaved(true)
    setTimeout(() => setConfigSaved(false), 2000)
    fetchModels()
  }, [config, fetchModels])

  // Execute API call
  const handleExecute = useCallback(async () => {
    if (!messages.some(m => m.content.trim()) || !selectedModel) return

    setLoading(true)
    setResponse(null)
    setStreamContent('')

    const startTime = Date.now()

    try {
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
      }
      if (config.apiKey) {
        headers['Authorization'] = `Bearer ${config.apiKey}`
      }

      const body = {
        model: selectedModel.id,
        messages: messages.filter(m => m.content.trim()),
        stream: streaming,
        max_tokens: 2048,
      }

      if (streaming) {
        // Handle streaming response
        const res = await fetch(`${config.baseUrl}/chat/completions`, {
          method: 'POST',
          headers,
          body: JSON.stringify(body),
        })

        if (!res.ok) {
          throw new Error(`API Error: ${res.status} ${res.statusText}`)
        }

        const reader = res.body?.getReader()
        const decoder = new TextDecoder()
        let fullContent = ''

        if (reader) {
          while (true) {
            const { done, value } = await reader.read()
            if (done) break

            const chunk = decoder.decode(value)
            const lines = chunk.split('\n')

            for (const line of lines) {
              if (line.startsWith('data: ') && line !== 'data: [DONE]') {
                try {
                  const data = JSON.parse(line.slice(6))
                  const content = data.choices?.[0]?.delta?.content || ''
                  fullContent += content
                  setStreamContent(fullContent)
                } catch {}
              }
            }
          }
        }

        setResponse({
          success: true,
          status_code: res.status,
          latency_ms: Date.now() - startTime,
          response: { content: fullContent },
        })
      } else {
        // Non-streaming request
        const res = await fetch(`${config.baseUrl}/chat/completions`, {
          method: 'POST',
          headers,
          body: JSON.stringify(body),
        })

        const data = await res.json()
        
        setResponse({
          success: res.ok,
          status_code: res.status,
          latency_ms: Date.now() - startTime,
          response: data,
          error: !res.ok ? (data.error?.message || JSON.stringify(data)) : undefined,
          input_tokens: data.usage?.prompt_tokens,
          output_tokens: data.usage?.completion_tokens,
          model: data.model,
        })
      }
    } catch (err: any) {
      setResponse({
        success: false,
        status_code: 0,
        latency_ms: Date.now() - startTime,
        error: err.message || 'Request failed',
      })
    } finally {
      setLoading(false)
    }
  }, [messages, selectedModel, config, streaming])

  const handleCopyResponse = useCallback(() => {
    const content = getAssistantContent()
    if (content) {
      navigator.clipboard.writeText(content)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }, [response, streamContent])

  const addMessage = useCallback(() => {
    setMessages(prev => [...prev, { role: 'user', content: '' }])
  }, [])

  const updateMessage = useCallback((index: number, field: keyof Message, value: string) => {
    setMessages(prev => {
      const updated = [...prev]
      updated[index] = { ...updated[index], [field]: value }
      return updated
    })
  }, [])

  const removeMessage = useCallback((index: number) => {
    setMessages(prev => prev.filter((_, i) => i !== index))
  }, [])

  const getAssistantContent = () => {
    if (streaming && streamContent) return streamContent
    if (!response?.response) return null
    try {
      const choices = response.response.choices
      if (choices?.[0]?.message?.content) {
        return choices[0].message.content
      }
      if (response.response.content) {
        return response.response.content
      }
    } catch {
      return JSON.stringify(response.response, null, 2)
    }
    return JSON.stringify(response.response, null, 2)
  }

  const getProviderFromModel = (modelId: string) => {
    if (modelId.includes('gpt') || modelId.includes('o1') || modelId.includes('o3')) return 'OpenAI'
    if (modelId.includes('claude')) return 'Anthropic'
    if (modelId.includes('gemini')) return 'Google'
    if (modelId.includes('llama') || modelId.includes('mixtral')) return 'Meta/Mistral'
    return 'Unknown'
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-xl bg-gradient-to-br from-blue-500/20 to-purple-500/20 border border-white/10">
            <Sparkles className="w-5 h-5 text-blue-400" />
          </div>
          <div>
            <h2 className="text-xl font-bold text-white">API Playground</h2>
            <p className="text-sm text-zinc-400">Test API calls interactively</p>
          </div>
        </div>
        <button
          onClick={() => setShowConfig(!showConfig)}
          className={`flex items-center gap-2 px-3 py-2 rounded-lg transition-all ${
            showConfig 
              ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30' 
              : 'bg-zinc-800/50 text-zinc-400 border border-white/5 hover:text-white'
          }`}
        >
          <Settings className="w-4 h-4" />
          Settings
        </button>
      </div>

      {/* Configuration Panel */}
      <AnimatePresence>
        {showConfig && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            className="overflow-hidden"
          >
            <div className="p-4 rounded-xl bg-zinc-900/50 border border-white/10 space-y-4">
              <div className="flex items-center gap-2 text-sm font-medium text-zinc-300">
                <Server className="w-4 h-4" />
                API Configuration
              </div>
              
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {/* Base URL */}
                <div className="space-y-2">
                  <label className="flex items-center gap-2 text-xs text-zinc-400">
                    <Globe className="w-3 h-3" />
                    Base URL
                  </label>
                  <input
                    type="text"
                    value={config.baseUrl}
                    onChange={(e) => setConfig(prev => ({ ...prev, baseUrl: e.target.value }))}
                    placeholder="http://127.0.0.1:8317/v1"
                    className="w-full px-3 py-2 rounded-lg bg-zinc-800 border border-white/10 text-white placeholder-zinc-500 focus:border-blue-500/50 focus:outline-none text-sm font-mono"
                  />
                </div>

                {/* API Key */}
                <div className="space-y-2">
                  <label className="flex items-center gap-2 text-xs text-zinc-400">
                    <Key className="w-3 h-3" />
                    API Key (optional)
                  </label>
                  <input
                    type="password"
                    value={config.apiKey}
                    onChange={(e) => setConfig(prev => ({ ...prev, apiKey: e.target.value }))}
                    placeholder="sk-..."
                    className="w-full px-3 py-2 rounded-lg bg-zinc-800 border border-white/10 text-white placeholder-zinc-500 focus:border-blue-500/50 focus:outline-none text-sm font-mono"
                  />
                </div>
              </div>

              <div className="flex items-center justify-between pt-2">
                <button
                  onClick={saveConfig}
                  className="flex items-center gap-2 px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 transition-colors text-white text-sm font-medium"
                >
                  {configSaved ? <Check className="w-4 h-4" /> : <CheckCircle className="w-4 h-4" />}
                  {configSaved ? 'Saved!' : 'Save & Fetch Models'}
                </button>
                
                <div className="flex items-center gap-2 text-xs text-zinc-500">
                  {loadingModels ? (
                    <>
                      <Loader2 className="w-3 h-3 animate-spin" />
                      Loading models...
                    </>
                  ) : modelsError ? (
                    <span className="text-amber-400">{modelsError}</span>
                  ) : (
                    <>
                      <CheckCircle className="w-3 h-3 text-green-400" />
                      {models.length} models available
                    </>
                  )}
                </div>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Input Section */}
        <div className="space-y-4">
          {/* Model Selector */}
          <div className="relative">
            <div className="flex items-center gap-2 mb-2">
              <span className="text-xs text-zinc-400">Model</span>
              <button
                onClick={fetchModels}
                disabled={loadingModels}
                className="p-1 rounded hover:bg-white/10 transition-colors text-zinc-500 hover:text-white"
                title="Refresh models"
              >
                <RefreshCw className={`w-3 h-3 ${loadingModels ? 'animate-spin' : ''}`} />
              </button>
            </div>
            
            <button
              onClick={() => setShowModelSelector(!showModelSelector)}
              className="w-full flex items-center justify-between px-4 py-3 rounded-xl bg-zinc-900/50 border border-white/10 hover:border-blue-500/30 transition-all"
            >
              <div className="flex items-center gap-3">
                <Zap className="w-4 h-4 text-blue-400" />
                {selectedModel ? (
                  <>
                    <span className="text-white font-mono text-sm">{selectedModel.id}</span>
                    <span className="text-xs px-2 py-0.5 rounded-full bg-zinc-800 text-zinc-400">
                      {getProviderFromModel(selectedModel.id)}
                    </span>
                  </>
                ) : (
                  <span className="text-zinc-500">Select a model...</span>
                )}
              </div>
              <ChevronDown className={`w-4 h-4 text-zinc-400 transition-transform ${showModelSelector ? 'rotate-180' : ''}`} />
            </button>

            <AnimatePresence>
              {showModelSelector && (
                <motion.div
                  initial={{ opacity: 0, y: -10 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -10 }}
                  className="absolute z-20 mt-2 w-full rounded-xl bg-zinc-900 border border-white/10 shadow-xl overflow-hidden max-h-64 overflow-y-auto"
                >
                  {models.length === 0 ? (
                    <div className="p-4 text-center text-zinc-500 text-sm">
                      No models found. Check your API configuration.
                    </div>
                  ) : (
                    models.map(model => (
                      <button
                        key={model.id}
                        onClick={() => {
                          setSelectedModel(model)
                          setShowModelSelector(false)
                        }}
                        className={`w-full flex items-center justify-between px-4 py-3 hover:bg-zinc-800/50 transition-all ${
                          model.id === selectedModel?.id ? 'bg-blue-500/10' : ''
                        }`}
                      >
                        <span className="text-white font-mono text-sm">{model.id}</span>
                        <span className="text-xs px-2 py-0.5 rounded-full bg-zinc-800 text-zinc-400">
                          {model.owned_by || getProviderFromModel(model.id)}
                        </span>
                      </button>
                    ))
                  )}
                </motion.div>
              )}
            </AnimatePresence>
          </div>

          {/* Streaming Toggle */}
          <label className="flex items-center gap-3 cursor-pointer">
            <div className={`relative w-10 h-5 rounded-full transition-colors ${streaming ? 'bg-blue-600' : 'bg-zinc-700'}`}>
              <div className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${streaming ? 'translate-x-5' : 'translate-x-0.5'}`} />
            </div>
            <span className="text-sm text-zinc-300">Stream response</span>
          </label>

          {/* Messages */}
          <div className="space-y-3">
            {messages.map((msg, index) => (
              <div key={index} className="space-y-2">
                <div className="flex items-center justify-between">
                  <select
                    value={msg.role}
                    onChange={(e) => updateMessage(index, 'role', e.target.value)}
                    className="text-xs px-2 py-1 rounded-lg bg-zinc-800 border border-white/10 text-zinc-300 focus:outline-none"
                  >
                    <option value="system">System</option>
                    <option value="user">User</option>
                    <option value="assistant">Assistant</option>
                  </select>
                  {messages.length > 1 && (
                    <button
                      onClick={() => removeMessage(index)}
                      className="text-xs text-zinc-500 hover:text-red-400 transition-colors"
                    >
                      Remove
                    </button>
                  )}
                </div>
                <textarea
                  value={msg.content}
                  onChange={(e) => updateMessage(index, 'content', e.target.value)}
                  placeholder={`Enter ${msg.role} message...`}
                  className="w-full min-h-[100px] px-4 py-3 rounded-xl bg-zinc-900/50 border border-white/10 text-white placeholder-zinc-500 focus:border-blue-500/50 focus:outline-none resize-none font-mono text-sm"
                />
              </div>
            ))}

            <button
              onClick={addMessage}
              className="text-sm text-blue-400 hover:text-blue-300 transition-colors"
            >
              + Add message
            </button>
          </div>

          {/* Execute Button */}
          <button
            onClick={handleExecute}
            disabled={loading || !messages.some(m => m.content.trim()) || !selectedModel}
            className="w-full flex items-center justify-center gap-2 px-4 py-3 rounded-xl bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-500 hover:to-purple-500 disabled:opacity-50 disabled:cursor-not-allowed transition-all text-white font-medium"
          >
            {loading ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                {streaming ? 'Streaming...' : 'Executing...'}
              </>
            ) : (
              <>
                <Play className="w-4 h-4" />
                Execute
              </>
            )}
          </button>
        </div>

        {/* Response Section */}
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium text-zinc-300">Response</span>
            {(response || streamContent) && (
              <button
                onClick={handleCopyResponse}
                className="flex items-center gap-1.5 text-xs text-zinc-400 hover:text-white transition-colors"
              >
                {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
                {copied ? 'Copied!' : 'Copy'}
              </button>
            )}
          </div>

          {/* Stats */}
          {response && (
            <div className="flex gap-3 flex-wrap text-xs">
              <div className={`flex items-center gap-1.5 px-2 py-1 rounded-lg ${
                response.success ? 'bg-emerald-500/10 text-emerald-400' : 'bg-red-500/10 text-red-400'
              }`}>
                {response.success ? <CheckCircle className="w-3 h-3" /> : <XCircle className="w-3 h-3" />}
                {response.success ? 'Success' : 'Failed'}
                <span className="text-zinc-500">({response.status_code})</span>
              </div>
              <div className="flex items-center gap-1.5 px-2 py-1 rounded-lg bg-zinc-800 text-zinc-300">
                <Clock className="w-3 h-3" />
                {response.latency_ms}ms
              </div>
              {(response.input_tokens || response.output_tokens) && (
                <div className="flex items-center gap-1.5 px-2 py-1 rounded-lg bg-zinc-800 text-zinc-300">
                  <Coins className="w-3 h-3" />
                  {response.input_tokens || 0} in / {response.output_tokens || 0} out
                </div>
              )}
              {response.model && (
                <div className="flex items-center gap-1.5 px-2 py-1 rounded-lg bg-zinc-800 text-zinc-300 font-mono">
                  {response.model}
                </div>
              )}
            </div>
          )}

          {/* Response Content */}
          <div className="relative min-h-[300px] rounded-xl bg-zinc-900/50 border border-white/10 overflow-hidden">
            {loading && streaming ? (
              <div className="p-4">
                <pre className="text-sm text-zinc-300 whitespace-pre-wrap font-mono">
                  {streamContent || <span className="text-zinc-500 animate-pulse">Waiting for response...</span>}
                </pre>
              </div>
            ) : loading ? (
              <div className="absolute inset-0 flex items-center justify-center">
                <Loader2 className="w-8 h-8 text-blue-400 animate-spin" />
              </div>
            ) : response ? (
              response.error ? (
                <div className="p-4 flex items-start gap-3">
                  <AlertCircle className="w-5 h-5 text-red-400 flex-shrink-0 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium text-red-400 mb-1">Error</p>
                    <pre className="text-xs text-zinc-400 whitespace-pre-wrap">{response.error}</pre>
                  </div>
                </div>
              ) : (
                <pre className="p-4 text-sm text-zinc-300 whitespace-pre-wrap overflow-auto max-h-[400px] font-mono">
                  {getAssistantContent()}
                </pre>
              )
            ) : (
              <div className="absolute inset-0 flex items-center justify-center text-zinc-500">
                <div className="text-center">
                  <Send className="w-8 h-8 mx-auto mb-2 opacity-50" />
                  <p className="text-sm">Execute a request to see the response</p>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

export default APIPlayground
