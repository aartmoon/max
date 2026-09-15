import { test, expect } from "@playwright/test";
test("resident creates an issue with a photo and follows status history", async ({
  page,
}) => {
  await page.goto("/");
  await page.screenshot({path:"test-results/home-mobile.png",fullPage:true});
  await page
    .getByRole("link", { name: "Сообщить о проблеме", exact: false })
    .first()
    .click();
  await page
    .getByLabel("Описание")
    .fill("Течёт труба в подъезде — браузерная проверка");
  await page
    .getByLabel("Фотография")
    .setInputFiles({
      name: "test.png",
      mimeType: "image/png",
      buffer: Buffer.from(
        "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aB1sAAAAASUVORK5CYII=",
        "base64",
      ),
    });
  await page.getByRole("button", { name: "Продолжить" }).click();
  await expect(
    page.getByRole("heading", { name: "Обращение сохранено" }),
  ).toBeVisible();
  await expect(page.getByText("PIPE_LEAK", { exact: true })).toBeVisible();
  await expect(
    page.getByRole("img", { name: "Фото к обращению" }),
  ).toBeVisible();
  await expect(page.getByRole("img", { name: "Фото к обращению" })).toHaveJSProperty("naturalWidth", 1);
  await page.screenshot({path:"test-results/request-mobile.png",fullPage:true});
  await page.reload();
  await expect(
    page.getByRole("img", { name: "Фото к обращению" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Следующий статус" }).click();
  await expect(page.locator(".timeline")).toContainText("Отправлено");
  await page.getByRole("link", { name: "← Мои обращения" }).click();
  await expect(
    page
      .getByRole("heading", {
        name: "Течёт труба в подъезде — браузерная проверка",
      })
      .first(),
  ).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
});
test("question and application forms are accessible", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("link", { name: "Задать вопрос" }).click();
  await expect(page.getByLabel("Ваш вопрос")).toBeVisible();
  await page.goto("/requests/new?kind=APPLICATION");
  await expect(
    page.getByRole("heading", { name: "Заявка в УК" }),
  ).toBeVisible();
});
test("server error is shown without losing input", async ({ page }) => {
  await page.goto("/requests/new");
  await page.getByLabel("Описание").fill("Не работает лифт");
  await page.route("**/api/requests", (route) =>
    route.fulfill({
      status: 503,
      contentType: "application/json",
      body: JSON.stringify({ error: "Сервер временно недоступен" }),
    }),
  );
  await page.getByRole("button", { name: "Продолжить" }).click();
  await expect(page.getByRole("alert")).toContainText(
    "Сервер временно недоступен",
  );
  await expect(page.getByLabel("Описание")).toHaveValue("Не работает лифт");
  await expect(page.getByRole("button", { name: "Продолжить" })).toBeEnabled();
});
