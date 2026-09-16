import { test, expect } from "@playwright/test";
const sdk = "https://st.max.ru/js/max-web-app.js";
const stub = `window.bridgeCalls={ready:0,visible:false,handlers:[],closing:false};
window.WebApp={initData:'start_param=house&hash=test',initDataUnsafe:{start_param:'house'},platform:'android',
ready(){window.bridgeCalls.ready++;window.bridgeCalls.rendered=!!document.querySelector('main')},
BackButton:{show(){window.bridgeCalls.visible=true},hide(){window.bridgeCalls.visible=false},onClick(fn){window.bridgeCalls.handlers.push(fn)},offClick(fn){window.bridgeCalls.handlers=window.bridgeCalls.handlers.filter(x=>x!==fn)}},
enableClosingConfirmation(){window.bridgeCalls.closing=true},disableClosingConfirmation(){window.bridgeCalls.closing=false}};`;
test("MAX SDK loads, ready runs after render and native back works for launch route", async ({
  page,
}) => {
  await page.route(sdk, (route) =>
    route.fulfill({ contentType: "application/javascript", body: stub }),
  );
  await page.goto("/max/");
  await expect(page).toHaveURL(/\/house$/);
  await expect
    .poll(() => page.evaluate(() => (window as any).bridgeCalls?.ready))
    .toBe(1);
  expect(await page.evaluate(() => (window as any).bridgeCalls.rendered)).toBe(
    true,
  );
  await expect
    .poll(() =>
      page.evaluate(() => (window as any).bridgeCalls.handlers.length),
    )
    .toBe(1);
  await page.evaluate(() => (window as any).bridgeCalls.handlers[0]());
  await expect(page).toHaveURL(/\/$/);
  await expect
    .poll(() => page.evaluate(() => (window as any).bridgeCalls.visible))
    .toBe(false);
  await page
    .getByRole("link", { name: "Создать заявку", exact: false })
    .first()
    .click();
  await page.getByLabel("Тип заявки").selectOption("EMERGENCY");
  await expect.poll(() => page.evaluate(() => (window as any).bridgeCalls.closing)).toBe(true);
  await page.getByLabel("Тип заявки").selectOption("APPLICATION");
  await expect.poll(() => page.evaluate(() => (window as any).bridgeCalls.closing)).toBe(false);
  await page.getByLabel("Описание").fill("В подъезде не работает лифт");
  await expect
    .poll(() => page.evaluate(() => (window as any).bridgeCalls.closing))
    .toBe(true);
  await page.getByLabel("Описание").fill("");
  await expect
    .poll(() => page.evaluate(() => (window as any).bridgeCalls.closing))
    .toBe(false);
  await page.evaluate(() => (window as any).bridgeCalls.handlers[0]());
  await expect(page).toHaveURL(/\/$/);
  expect(
    await page.evaluate(() => (window as any).bridgeCalls.handlers.length),
  ).toBe(0);
});
test("website works if SDK is unavailable", async ({ page }) => {
  await page.route(sdk, (route) => route.abort());
  await page.goto("/max/");
  await page
    .getByRole("link", { name: "Создать заявку", exact: false })
    .first()
    .click();
  await expect(
    page.getByRole("heading", { name: "Новая заявка" }),
  ).toBeVisible();
});
test("late SDK is used and unknown launch parameter cannot navigate externally", async ({
  page,
}) => {
  let release!: () => void;
  const gate = new Promise<void>((resolve) => {
    release = resolve;
  });
  await page.route(sdk, async (route) => {
    await gate;
    await route.fulfill({
      contentType: "application/javascript",
      body: stub.replace(
        "start_param:'house'",
        "start_param:'https://evil.example'",
      ),
    });
  });
  await page.goto("/max/", { waitUntil: "domcontentloaded" });
  await expect(
    page.getByRole("link", { name: "Создать заявку", exact: false }).first(),
  ).toBeVisible();
  release();
  await expect
    .poll(() => page.evaluate(() => (window as any).bridgeCalls?.ready))
    .toBe(1);
  await expect(page).toHaveURL(/\/$/);
});
