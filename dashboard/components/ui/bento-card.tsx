'use client'

import { cn } from "@/lib/utils"
import { motion, HTMLMotionProps } from "framer-motion"
import { forwardRef, useRef, useState, useEffect, ReactNode } from "react"

interface BentoCardProps extends Omit<HTMLMotionProps<"div">, "children"> {
  children: ReactNode
  className?: string
  spotlight?: boolean
  glow?: "blue" | "purple" | "cyan" | "emerald" | "amber" | "rose" | "none"
  size?: "sm" | "md" | "lg" | "xl"
  hover?: boolean
}

const glowColors = {
  blue: "from-blue-500/20 via-blue-500/5 to-transparent",
  purple: "from-purple-500/20 via-purple-500/5 to-transparent",
  cyan: "from-cyan-500/20 via-cyan-500/5 to-transparent",
  emerald: "from-emerald-500/20 via-emerald-500/5 to-transparent",
  amber: "from-amber-500/20 via-amber-500/5 to-transparent",
  rose: "from-rose-500/20 via-rose-500/5 to-transparent",
  none: ""
}

const sizeClasses = {
  sm: "p-4",
  md: "p-5",
  lg: "p-6",
  xl: "p-8"
}

const BentoCard = forwardRef<HTMLDivElement, BentoCardProps>(({
  children,
  className,
  spotlight = true,
  glow = "blue",
  size = "md",
  hover = true,
  ...props
}, ref) => {
  const cardRef = useRef<HTMLDivElement>(null)
  const [mousePosition, setMousePosition] = useState({ x: 0, y: 0 })
  const [isHovered, setIsHovered] = useState(false)

  useEffect(() => {
    const card = cardRef.current
    if (!card || !spotlight) return

    const handleMouseMove = (e: MouseEvent) => {
      const rect = card.getBoundingClientRect()
      const x = e.clientX - rect.left
      const y = e.clientY - rect.top
      setMousePosition({ x, y })
    }

    card.addEventListener('mousemove', handleMouseMove)
    return () => card.removeEventListener('mousemove', handleMouseMove)
  }, [spotlight])

  return (
    <motion.div
      ref={(node) => {
        (cardRef as any).current = node
        if (typeof ref === 'function') ref(node)
        else if (ref) ref.current = node
      }}
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      whileHover={hover ? { y: -4, transition: { duration: 0.2, ease: "easeOut" } } : undefined}
      onHoverStart={() => setIsHovered(true)}
      onHoverEnd={() => setIsHovered(false)}
      className={cn(
        "relative overflow-hidden rounded-2xl",
        "bg-zinc-900/40 backdrop-blur-xl",
        "border border-white/[0.08]",
        "transition-all duration-300",
        hover && "hover:border-white/[0.15] hover:bg-zinc-900/60",
        sizeClasses[size],
        className
      )}
      style={{
        '--mouse-x': `${mousePosition.x}px`,
        '--mouse-y': `${mousePosition.y}px`
      } as any}
      {...props}
    >
      {spotlight && (
        <div
          className="pointer-events-none absolute inset-0 opacity-0 transition-opacity duration-300"
          style={{
            opacity: isHovered ? 1 : 0,
            background: `radial-gradient(600px circle at ${mousePosition.x}px ${mousePosition.y}px, rgba(255,255,255,0.04), transparent 40%)`
          }}
        />
      )}
      
      {glow !== "none" && (
        <div className={cn(
          "absolute -top-1/2 -right-1/2 w-full h-full rounded-full blur-3xl opacity-0 transition-opacity duration-500",
          "bg-gradient-radial",
          glowColors[glow],
          isHovered && "opacity-100"
        )} />
      )}

      <div className="relative z-10">
        {children}
      </div>
    </motion.div>
  )
})
BentoCard.displayName = "BentoCard"

interface BentoGridProps {
  children: ReactNode
  className?: string
  cols?: 1 | 2 | 3 | 4
}

function BentoGrid({ children, className, cols = 4 }: BentoGridProps) {
  const colsClass = {
    1: "grid-cols-1",
    2: "grid-cols-1 md:grid-cols-2",
    3: "grid-cols-1 md:grid-cols-2 lg:grid-cols-3",
    4: "grid-cols-1 md:grid-cols-2 lg:grid-cols-4"
  }

  return (
    <div className={cn("grid gap-4", colsClass[cols], className)}>
      {children}
    </div>
  )
}

export { BentoCard, BentoGrid }
