import 'package:flutter/material.dart';

/// Definição centralizada do tema visual oficial do SysAP Mobile (Material 3).
///
/// Identidade: "Laboratório de Performance + Centro de Comando Comercial".
/// Cores da marca: Fundo Obsidiana (#080A0C) e Dourado Artur Performance (#D4AE29).
class TemaSysAP {
  const TemaSysAP._();

  // Cores canônicas da identidade visual
  static const Color corFundoObsidiana = Color(0xFF080A0C);
  static const Color corSuperficieEscura = Color(0xFF12161A);
  static const Color corSuperficieElevada = Color(0xFF181D22);
  static const Color corDouradoPrincipal = Color(0xFFD4AE29);
  static const Color corDouradoInteracao = Color(0xFFEAC855);
  static const Color corBordaTatica = Color(0xFF262C33);
  static const Color corTextoClaro = Color(0xFFFFFFFF);
  static const Color corTextoSecundario = Color(0xFFA1A1AA);
  static const Color corErro = Color(0xFFFF5252);
  static const Color corSucesso = Color(0xFF4CAF50);
  static const Color corAlerta = Color(0xFFFFC107);

  /// Tema claro neutro baseado em Material 3 com acentos dourados.
  static ThemeData get temaClaro {
    final esquema = ColorScheme.fromSeed(
      seedColor: corDouradoPrincipal,
      brightness: Brightness.light,
      primary: corDouradoPrincipal,
      error: corErro,
    );

    return ThemeData(
      useMaterial3: true,
      colorScheme: esquema,
      scaffoldBackgroundColor: const Color(0xFFF8FAFC),
      appBarTheme: AppBarTheme(
        centerTitle: true,
        backgroundColor: esquema.surface,
        foregroundColor: esquema.onSurface,
        elevation: 0,
      ),
      cardTheme: CardThemeData(
        elevation: 0,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(8),
          side: BorderSide(color: esquema.outlineVariant),
        ),
      ),
    );
  }

  /// Tema escuro oficial do SysAP (Obsidiana + Dourado Artur Performance).
  static ThemeData get temaEscuro {
    final esquema = ColorScheme.fromSeed(
      seedColor: corDouradoPrincipal,
      brightness: Brightness.dark,
      primary: corDouradoPrincipal,
      surface: corSuperficieEscura,
      error: corErro,
    );

    return ThemeData(
      useMaterial3: true,
      colorScheme: esquema,
      scaffoldBackgroundColor: corFundoObsidiana,
      appBarTheme: const AppBarTheme(
        centerTitle: true,
        backgroundColor: corFundoObsidiana,
        foregroundColor: corTextoClaro,
        elevation: 0,
      ),
      cardTheme: CardThemeData(
        color: corSuperficieEscura,
        elevation: 0,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(8),
          side: const BorderSide(color: corBordaTatica),
        ),
      ),
    );
  }
}
