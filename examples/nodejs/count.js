async function main() {
  for (let counter = 1; counter <= 10; counter++) {
    await new Promise((resolve) => setTimeout(resolve, 500));
    process.stdout.write(`${counter}\n`);
  }
}

main();
