package wiki

import (
	"bufio"
	"errors"
	"log"
	"regexp"
	"strings"
)

type Parser struct {
	report       *Report
	header       string
	subheader    string
	currentArray *[]string
	parser       func(line string)
}

func (parser *Parser) reset() {
	parser.header = ""
	parser.subheader = ""
	parser.currentArray = nil
	parser.parser = nil
}

func (parser *Parser) setHeader(header string, parserFunc func(line string)) {
	parser.header = header
	parser.subheader = ""
	parser.currentArray = nil
	parser.parser = parserFunc
}

func (parser *Parser) setSubheader(subheader string) {
	parser.subheader = strings.TrimSpace(subheader)
	parser.currentArray = nil
}

func (parser *Parser) parseHolidays(line string) {
	line = strings.Trim(line, ".;— ")
	if strings.HasPrefix(line, "См. также:") {
		return
	}
	if parser.subheader == "" {
		parser.report.HolidaysInt = append(parser.report.HolidaysInt, line)
		return
	} else if parser.currentArray == nil && parser.subheader != rlgHolidaysSubheader {
		switch parser.subheader {
		case intHolidaysSubheader:
			parser.currentArray = &parser.report.HolidaysInt
		case locHolidaysSubheader:
			parser.currentArray = &parser.report.HolidaysLoc
		case profHolidaysSubheader:
			parser.currentArray = &parser.report.HolidaysProf
		case nameDaysSubheader:
			parser.parser = parser.parseNamedays
			parser.parser(line)
			return
		default:
			parser.subheader = ""
			return
		}
	} else if parser.subheader == rlgHolidaysSubheader {
		if line == "Христианские" {
			return
		}
		extraLinkMatch := regexp.MustCompile("Примечание: указано для невисокосных лет, в високосные годы список иной, см. \\d+ .*?\\.|\\(.*, см. \\d+ .*?\\)")
		orthRegex := regexp.MustCompile("Православ(ие|ные)( (\\(|.*)Русская Православная Церковь(\\)|.*))?|В .*[Пп]равосл.* церкв(и|ях):?|(\\(|.*)Русская Православная Церковь(\\)|.*)")
		cathRegex := regexp.MustCompile("Католи(цизм|ческие|чество)|В [Кк]атолич.* церкв(и|ях)")
		othersRegex := regexp.MustCompile("Зороастризм|Другие конфессии|В католичестве и протестантстве|:?Славянские праздники:?|Ислам(ские|.?)|В Древневосточных церквях|Буддизм")
		bahaiRegex := regexp.MustCompile("Бахаи")
		switch {
		case extraLinkMatch.MatchString(line):
			line = parser.splitLineWithHeader(extraLinkMatch, line, nil)
		case orthRegex.MatchString(line):
			newItem := ReligiousHolidayDescr{GroupAbbr: "правосл."}
			parser.report.HolidaysRlg.Holidays = append(parser.report.HolidaysRlg.Holidays, &newItem)
			line = parser.splitLineWithHeader(orthRegex, line, &newItem.Descriptions)
		case cathRegex.MatchString(line):
			newItem := ReligiousHolidayDescr{GroupAbbr: "катол."}
			parser.report.HolidaysRlg.Holidays = append(parser.report.HolidaysRlg.Holidays, &newItem)
			line = parser.splitLineWithHeader(cathRegex, line, &newItem.Descriptions)
		case othersRegex.MatchString(line):
			newItem := ReligiousHolidayDescr{}
			parser.report.HolidaysRlg.Holidays = append(parser.report.HolidaysRlg.Holidays, &newItem)
			line = parser.splitLineWithHeader(othersRegex, line, &newItem.Descriptions)
		case bahaiRegex.MatchString(line):
			newItem := ReligiousHolidayDescr{GroupAbbr: "бахаи"}
			parser.report.HolidaysRlg.Holidays = append(parser.report.HolidaysRlg.Holidays, &newItem)
			line = parser.splitLineWithHeader(bahaiRegex, line, &newItem.Descriptions)
		case parser.currentArray == nil:
			newItem := ReligiousHolidayDescr{}
			parser.report.HolidaysRlg.Holidays = append(parser.report.HolidaysRlg.Holidays, &newItem)
			parser.currentArray = &newItem.Descriptions
		}
		reApostle := regexp.MustCompile("память апостол.*")
		reMemorial := regexp.MustCompile("^[Пп]амять .*")

		if has := reMemorial.MatchString(line); has {
			if has = reApostle.MatchString(line); !has {
				return
			}
		}
	}
	if parser.currentArray == nil {
		log.Print("Error parsing:", line)
		return
	}
	if line == "" {
		return
	}
	*parser.currentArray = append(*parser.currentArray, line)
}

func (parser *Parser) splitLineWithHeader(headerRegexp *regexp.Regexp, line string, filled *[]string) string {
	index := headerRegexp.FindStringIndex(line)
	if index[0] == 0 {
		if filled != nil {
			parser.currentArray = filled
		}
		line = headerRegexp.Split(line, 2)[1]
	} else {
		lines := headerRegexp.Split(line, 2)
		parser.parseHolidays(lines[0])
		if filled != nil {
			parser.currentArray = filled
		}
		line = lines[1]
	}
	return line
}

func (parser *Parser) parseNamedays(line string) {
	line = strings.Trim(line, ".;— ")
	reAs := regexp.MustCompile("также:")
	if has := reAs.MatchString(line); has {
		lines := reAs.Split(line, 2)
		for _, l := range lines {
			l = strings.TrimSpace(l)
			if l != "" {
				parser.report.NameDays = append(parser.report.NameDays, l)
			}
		}
		return
	}
	reAs = regexp.MustCompile("и производные:")
	if has := reAs.MatchString(line); has {
		line = reAs.Split(line, 2)[0]
	}
	parser.report.NameDays = append(parser.report.NameDays, strings.TrimSpace(line))
}

func (parser *Parser) parseOmens(line string) {
	if parser.currentArray == nil {
		parser.currentArray = &parser.report.Omens
	}

	if len(*parser.currentArray) != 0 {
		parser.appendOmens(line, false)
	} else {
		parser.appendOmens(line, true)
	}
}

func (parser *Parser) appendOmens(line string, split bool) {
	if !split {
		line = strings.Replace(line, "* ", "", -1);
		line = strings.Trim(line, "…,. ")
		if line == "" {
			return
		}
		*parser.currentArray = append(*parser.currentArray, line)
		return
	}

	lines := strings.Split(line, ".")
	for _, l := range lines {
		line = strings.Trim(l, "…,. ")
		if line == "" {
			continue
		}
		*parser.currentArray = append(*parser.currentArray, line)
	}
}

func Parse(fullReport string) (Report, error) {
	report := Report{}
	if fullReport == "" {
		return report, errors.New("empty report")
	}
	scanner := bufio.NewScanner(strings.NewReader(fullReport))
	parser := Parser{report: &report}

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "== ") && strings.HasSuffix(line, " =="):
			switch header := strings.TrimSpace(strings.Trim(line, "==")); header {
			case holidaysHeader, "Праздники":
				parser.setHeader(header, parser.parseHolidays)
			case "События", "Родились", "Скончались":
				parser.reset()
			case "Приметы", "Народный календарь", "Народный календарь и приметы", "Народный календарь, приметы", "Народный календарь, приметы и фольклор Руси":
				parser.setHeader(header, parser.parseOmens)
			default:
				parser.reset()
				log.Print("Extra header:", header)
			}
		case strings.HasPrefix(line, "=== ") && strings.HasSuffix(line, " ==="):
			parser.setSubheader(strings.Trim(line, "==="))
		case strings.HasPrefix(line, "==== ") && strings.HasSuffix(line, " ===="):
			parser.parser(strings.Trim(line, "===="))
		case line == "":
			continue
		default:
			if parser.parser == nil {
				continue
			}
			parser.parser(strings.TrimSpace(line))
		}
	}
	return report, nil
}