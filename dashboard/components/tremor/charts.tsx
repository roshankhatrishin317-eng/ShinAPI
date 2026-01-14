'use client'

import React from 'react'
import {
  AreaChart as TremorAreaChart,
  BarChart as TremorBarChart,
  DonutChart as TremorDonutChart,
  Card,
  Metric,
  Text,
  Flex,
  BadgeDelta,
  ProgressBar,
  SparkAreaChart,
  Legend,
  CategoryBar,
} from '@tremor/react'

interface ChartData {
  time?: string
  date?: string
  name?: string
  value?: number
  [key: string]: any
}

// KPI Card with sparkline
export function KPICard({
  title,
  metric,
  delta,
  deltaType = 'unchanged',
  sparklineData = [],
  color = 'blue',
  suffix = '',
  prefix = '',
}: {
  title: string
  metric: number | string
  delta?: string
  deltaType?: 'increase' | 'decrease' | 'unchanged' | 'moderateIncrease' | 'moderateDecrease'
  sparklineData?: number[]
  color?: string
  suffix?: string
  prefix?: string
}) {
  return (
    <Card className="bg-zinc-900/50 border-white/10 ring-0">
      <Flex alignItems="start">
        <div className="truncate">
          <Text className="text-zinc-400">{title}</Text>
          <Metric className="text-white truncate">
            {prefix}{typeof metric === 'number' ? metric.toLocaleString() : metric}{suffix}
          </Metric>
        </div>
        {delta && (
          <BadgeDelta deltaType={deltaType} size="xs">
            {delta}
          </BadgeDelta>
        )}
      </Flex>
      {sparklineData.length > 0 && (
        <SparkAreaChart
          data={sparklineData.map((v, i) => ({ index: i, value: v }))}
          categories={['value']}
          index="index"
          colors={[color as any]}
          className="h-8 mt-4"
          curveType="monotone"
        />
      )}
    </Card>
  )
}

// Progress KPI Card
export function ProgressKPICard({
  title,
  metric,
  target,
  progress,
  color = 'blue',
}: {
  title: string
  metric: string | number
  target?: string
  progress: number
  color?: string
}) {
  return (
    <Card className="bg-zinc-900/50 border-white/10 ring-0">
      <Text className="text-zinc-400">{title}</Text>
      <Metric className="text-white">{metric}</Metric>
      <Flex className="mt-4">
        <Text className="text-zinc-500 truncate">{`${progress}% of ${target || 'target'}`}</Text>
      </Flex>
      <ProgressBar value={progress} color={color as any} className="mt-2" />
    </Card>
  )
}

// Area Chart
export function AreaChartComponent({
  data,
  categories,
  index = 'time',
  colors = ['blue', 'violet'],
  title,
  showLegend = true,
  height = 'h-72',
}: {
  data: ChartData[]
  categories: string[]
  index?: string
  colors?: string[]
  title?: string
  showLegend?: boolean
  height?: string
}) {
  return (
    <Card className="bg-zinc-900/50 border-white/10 ring-0">
      {title && <Text className="text-zinc-300 font-medium mb-4">{title}</Text>}
      <TremorAreaChart
        className={height}
        data={data}
        index={index}
        categories={categories}
        colors={colors as any}
        showAnimation
        showLegend={showLegend}
        curveType="monotone"
        yAxisWidth={48}
      />
    </Card>
  )
}

// Bar Chart
export function BarChartComponent({
  data,
  categories,
  index = 'name',
  colors = ['blue'],
  title,
  layout = 'vertical',
  height = 'h-72',
}: {
  data: ChartData[]
  categories: string[]
  index?: string
  colors?: string[]
  title?: string
  layout?: 'vertical' | 'horizontal'
  height?: string
}) {
  return (
    <Card className="bg-zinc-900/50 border-white/10 ring-0">
      {title && <Text className="text-zinc-300 font-medium mb-4">{title}</Text>}
      <TremorBarChart
        className={height}
        data={data}
        index={index}
        categories={categories}
        colors={colors as any}
        showAnimation
        layout={layout}
        yAxisWidth={48}
      />
    </Card>
  )
}

// Donut Chart
export function DonutChartComponent({
  data,
  category = 'value',
  index = 'name',
  colors,
  title,
  variant = 'donut',
  showLabel = true,
}: {
  data: ChartData[]
  category?: string
  index?: string
  colors?: string[]
  title?: string
  variant?: 'donut' | 'pie'
  showLabel?: boolean
}) {
  const defaultColors = ['blue', 'violet', 'cyan', 'emerald', 'amber', 'rose']
  return (
    <Card className="bg-zinc-900/50 border-white/10 ring-0">
      {title && <Text className="text-zinc-300 font-medium mb-4">{title}</Text>}
      <TremorDonutChart
        className="h-52"
        data={data}
        category={category}
        index={index}
        colors={(colors || defaultColors) as any}
        variant={variant}
        showAnimation
        label={showLabel ? data.reduce((acc, item) => acc + (item[category] || 0), 0).toLocaleString() : undefined}
      />
      <Legend
        categories={data.map(d => d[index] as string)}
        colors={(colors || defaultColors) as any}
        className="mt-4"
      />
    </Card>
  )
}

// Category Bar (for showing distribution)
export function CategoryBarComponent({
  values,
  colors = ['emerald', 'amber', 'rose'],
  labels,
  title,
  showLabels = true,
}: {
  values: number[]
  colors?: string[]
  labels?: string[]
  title?: string
  showLabels?: boolean
}) {
  return (
    <Card className="bg-zinc-900/50 border-white/10 ring-0">
      {title && <Text className="text-zinc-300 font-medium mb-4">{title}</Text>}
      <CategoryBar
        values={values}
        colors={colors as any}
        className="mt-2"
        showLabels={showLabels}
      />
      {labels && (
        <Legend
          categories={labels}
          colors={colors as any}
          className="mt-4"
        />
      )}
    </Card>
  )
}
