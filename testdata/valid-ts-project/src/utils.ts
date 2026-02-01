/**
 * Utility functions for data processing.
 */

import { User } from "./index";

// Simple one-liner function (complexity 1)
export function add(a: number, b: number): number {
  return a + b;
}

// Complex function with multiple branches (complexity > 5)
export function processData(items: unknown[], mode: string): string[] {
  const results: string[] = [];

  for (const item of items) {
    if (typeof item === "string") {
      if (item.length > 0) {
        results.push(item.toUpperCase());
      } else {
        results.push("EMPTY");
      }
    } else if (typeof item === "number") {
      if (item > 100 || item < -100) {
        results.push("OUT_OF_RANGE");
      } else {
        results.push(String(item));
      }
    } else {
      results.push("UNKNOWN");
    }
  }

  switch (mode) {
    case "sort":
      results.sort();
      break;
    case "reverse":
      results.reverse();
      break;
    case "unique":
      return [...new Set(results)];
    default:
      break;
  }

  return results;
}

// Class with methods
export class DataProcessor {
  private items: string[];
  private threshold: number;

  constructor(threshold: number) {
    this.items = [];
    this.threshold = threshold;
  }

  addItem(item: string): void {
    if (item && item.length > 0) {
      this.items.push(item);
    }
  }

  process(): string[] {
    const filtered: string[] = [];
    for (const item of this.items) {
      if (item.length > this.threshold) {
        filtered.push(item);
      }
    }
    return filtered;
  }

  getStats(): { count: number; avgLength: number } {
    const count = this.items.length;
    if (count === 0) {
      return { count: 0, avgLength: 0 };
    }
    let totalLength = 0;
    for (const item of this.items) {
      totalLength += item.length;
    }
    return { count, avgLength: totalLength / count };
  }
}

// Arrow function assigned to const
export const multiply = (a: number, b: number): number => a * b;

// Arrow function with complexity
export const categorize = (value: number): string => {
  if (value < 0) {
    return "negative";
  } else if (value === 0) {
    return "zero";
  } else if (value < 10) {
    return "small";
  } else {
    return "large";
  }
};

// Dead export: not imported by any other file in this project
export function unusedHelper(x: number): number {
  return x * 2 + 1;
}

// Another dead export
export class UnusedProcessor {
  run(): void {
    console.log("unused");
  }
}

// Function using ternary and nullish coalescing (for complexity testing)
export function formatUser(user: User | null): string {
  const name = user?.name ?? "Anonymous";
  const greeting = user?.age !== undefined ? `${name} (${user.age})` : name;
  return greeting;
}
