'use client'

import { motion, AnimatePresence } from 'framer-motion'
import { cn } from '@/lib/utils'
import { 
  Radio, Cpu, Database, CheckCircle, XCircle, 
  ArrowRight, Zap
} from 'lucide-react'

interface PipelineProps {
  rpm: number
  tps: number
  tpm: number
  totalSuccess: number
  totalFailed: number
}

function formatNumber(num: number): string {
  if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
  return Math.round(num).toString()
}

const stages = [
  { key: 'incoming', icon: Radio, label: 'Incoming', color: 'from-blue-500 to-blue-600' },
  { key: 'processing', icon: Cpu, label: 'Processing', color: 'from-purple-500 to-purple-600' },
  { key: 'tokens', icon: Database, label: 'Tokens', color: 'from-cyan-500 to-cyan-600' },
  { key: 'success', icon: CheckCircle, label: 'Success', color: 'from-emerald-500 to-emerald-600' },
  { key: 'failed', icon: XCircle, label: 'Failed', color: 'from-rose-500 to-rose-600' },
]

function PipelineStage({ 
  icon: Icon, 
  label, 
  value, 
  color, 
  isActive,
  delay 
}: { 
  icon: React.ElementType
  label: string
  value: string
  color: string
  isActive: boolean
  delay: number
}) {
  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.8, y: 20 }}
      animate={{ opacity: 1, scale: 1, y: 0 }}
      transition={{ delay, type: "spring", stiffness: 200 }}
      className="flex flex-col items-center group cursor-pointer"
    >
      <div className="relative">
        <motion.div
          className={cn(
            "w-16 h-16 rounded-2xl flex items-center justify-center",
            "bg-gradient-to-br shadow-lg transition-transform",
            "group-hover:scale-110",
            color
          )}
          whileHover={{ rotate: 5 }}
        >
          <Icon className="w-7 h-7 text-white" />
        </motion.div>
        
        {isActive && (
          <motion.div
            className={cn("absolute inset-0 rounded-2xl bg-gradient-to-br", color)}
            animate={{ 
              scale: [1, 1.3, 1],
              opacity: [0.5, 0, 0.5]
            }}
            transition={{ duration: 2, repeat: Infinity }}
          />
        )}
      </div>
      
      <motion.p 
        className="mt-3 text-[11px] text-zinc-500 uppercase tracking-widest font-medium"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: delay + 0.1 }}
      >
        {label}
      </motion.p>
      
      <motion.p 
        className="text-lg font-bold text-white"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: delay + 0.2 }}
      >
        {value}
      </motion.p>
    </motion.div>
  )
}

function FlowConnector({ isActive }: { isActive: boolean }) {
  return (
    <div className="flex-1 max-w-20 h-1 bg-zinc-800/50 rounded-full relative overflow-hidden mx-2">
      {isActive && (
        <motion.div
          className="absolute inset-y-0 w-8 bg-gradient-to-r from-transparent via-white/30 to-transparent"
          animate={{ x: [-32, 80] }}
          transition={{ 
            duration: 1.2, 
            repeat: Infinity, 
            ease: "linear"
          }}
        />
      )}
      <motion.div
        className="absolute inset-0 bg-gradient-to-r from-blue-500/50 via-purple-500/50 to-cyan-500/50"
        initial={{ scaleX: 0 }}
        animate={{ scaleX: isActive ? 1 : 0.3 }}
        style={{ transformOrigin: 'left' }}
        transition={{ duration: 0.5 }}
      />
    </div>
  )
}

export function RequestPipeline({ rpm, tps, tpm, totalSuccess, totalFailed }: PipelineProps) {
  const values = [
    formatNumber(rpm) + '/m',
    tps.toFixed(1) + '/s',
    formatNumber(tpm),
    formatNumber(totalSuccess),
    formatNumber(totalFailed)
  ]

  const isActive = [rpm > 0, tps > 0, tpm > 0, true, totalFailed > 0]

  return (
    <div className="relative">
      <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
        <motion.div
          className="w-full h-full bg-gradient-to-r from-blue-500/5 via-purple-500/5 to-cyan-500/5 rounded-3xl blur-2xl"
          animate={{ opacity: [0.3, 0.5, 0.3] }}
          transition={{ duration: 4, repeat: Infinity }}
        />
      </div>

      <div className="relative flex items-center justify-between py-6 px-4">
        {stages.map((stage, i) => (
          <motion.div 
            key={stage.key}
            className="flex items-center"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
          >
            <PipelineStage
              icon={stage.icon}
              label={stage.label}
              value={values[i]}
              color={stage.color}
              isActive={isActive[i]}
              delay={i * 0.1}
            />
            
            {i < stages.length - 1 && (
              <FlowConnector isActive={isActive[i]} />
            )}
          </motion.div>
        ))}
      </div>
    </div>
  )
}
