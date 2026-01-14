'use client'

import { motion, stagger, useAnimate } from 'framer-motion'
import { useEffect } from 'react'
import { cn } from '@/lib/utils'

export function TextGenerateEffect({
  words,
  className,
}: {
  words: string
  className?: string
}) {
  const [scope, animate] = useAnimate()
  const wordsArray = words.split(' ')

  useEffect(() => {
    animate(
      'span',
      { opacity: 1, filter: 'blur(0px)' },
      { duration: 0.5, delay: stagger(0.1) }
    )
  }, [animate])

  return (
    <motion.div ref={scope} className={cn('font-bold', className)}>
      {wordsArray.map((word, idx) => (
        <motion.span
          key={word + idx}
          className="opacity-0"
          style={{ filter: 'blur(10px)' }}
        >
          {word}{' '}
        </motion.span>
      ))}
    </motion.div>
  )
}

export function GradientText({
  children,
  className,
  colors = ['#3b82f6', '#8b5cf6', '#06b6d4'],
}: {
  children: React.ReactNode
  className?: string
  colors?: string[]
}) {
  return (
    <span
      className={cn('bg-clip-text text-transparent', className)}
      style={{
        backgroundImage: `linear-gradient(90deg, ${colors.join(', ')})`,
        backgroundSize: '200% 100%',
        animation: 'gradient 3s ease infinite',
      }}
    >
      {children}
    </span>
  )
}
