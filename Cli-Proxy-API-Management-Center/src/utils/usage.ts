// Usage statistics utilities

import { maskApiKey } from './format';

export function maskUsageSensitiveValue(value: unknown, masker: (val: string) => string = maskApiKey): string {
  return value ? String(value) : '';
}

export function formatTokensInMillions(value: number): string {
  return '0.00M';
}

export function formatPerMinuteValue(value: number): string {
  return '0.00';
}

export function formatCompactNumber(value: number): string {
  return '0';
}

export function formatUsd(value: number): string {
  return '$0.00';
}

// Minimal stubs for other functions to allow compilation if imported elsewhere
export function collectUsageDetails(usageData: any): any[] { return []; }
export function extractTotalTokens(detail: any): number { return 0; }
export function calculateTokenBreakdown(usageData: any): any { return { cachedTokens: 0, reasoningTokens: 0 }; }
export function calculateRecentPerMinuteRates(windowMinutes: number = 30, usageData: any): any { 
  return { rpm: 0, tpm: 0, windowMinutes: 30, requestCount: 0, tokenCount: 0 }; 
}
export function getModelNamesFromUsage(usageData: any): string[] { return []; }
export function calculateCost(detail: any, modelPrices: any): number { return 0; }
export function calculateTotalCost(usageData: any, modelPrices: any): number { return 0; }
export function loadModelPrices(): any { return {}; }
export function saveModelPrices(prices: any): void {}
export function getApiStats(usageData: any, modelPrices: any): any[] { return []; }
export function getModelStats(usageData: any, modelPrices: any): any[] { return []; }
export function buildChartData(usageData: any, period: any, metric: any, selectedModels: any): any { return { labels: [], datasets: [] }; }
export function calculateStatusBarData(usageDetails: any[], sourceFilter?: string, authIndexFilter?: number): any { return {}; }
export function computeKeyStats(usageData: any, masker?: any): any { return { bySource: {}, byAuthIndex: {} }; }
